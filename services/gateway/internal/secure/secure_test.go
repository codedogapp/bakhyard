package secure_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/gateway/internal/secure"
)

// The app's policy (docs/plan.md §4.4), and the stricter one of a vault file (decision 92).
const (
	appPolicy = "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; " +
		"font-src 'self' data:; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; " +
		"base-uri 'none'"
	vaultPolicy = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
)

func TestHeadersAddTheSecurityHeaders(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"a body": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "hello")
		},
		"a status alone": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		"nothing": func(http.ResponseWriter, *http.Request) {},
		"an error": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		},
	} {
		t.Run(name, func(t *testing.T) {
			expect(t, recorded(t, handler).Result().Header, headers(appPolicy))
		})
	}
}

func TestHeadersGoOutWithAFlush(t *testing.T) {
	rec := recorded(t, func(w http.ResponseWriter, _ *http.Request) {
		err := http.NewResponseController(w).Flush()
		if err != nil {
			t.Errorf("flush: %v", err)
		}
	})

	if !rec.Flushed {
		t.Error("the response wasn't flushed")
	}

	expect(t, rec.Result().Header, headers(appPolicy))
}

func TestHeadersKeepAResponsesOwnValue(t *testing.T) {
	for key := range headers(appPolicy) {
		t.Run(key, func(t *testing.T) {
			rec := recorded(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(key, "own")
				w.WriteHeader(http.StatusOK)
			})

			want := headers(appPolicy)
			want[key] = "own"
			expect(t, rec.Result().Header, want)
		})
	}
}

func TestHeadersLeaveAProxiedServicesOwnHeadersAlone(t *testing.T) {
	header := proxiedHeader(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Security-Policy", vaultPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.WriteString(w, "# A note")
	})

	expect(t, header, headers(vaultPolicy))
}

func TestHeadersOutlastAnEarlyHint(t *testing.T) {
	// ReverseProxy clears the header map after it passes a 1xx on.
	header := proxiedHeader(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", "</app.js>; rel=preload; as=script")
		w.WriteHeader(http.StatusEarlyHints)
		_, _ = io.WriteString(w, "hello")
	})

	expect(t, header, headers(appPolicy))
}

func TestHeadersLetAResponseStream(t *testing.T) {
	release := make(chan struct{})

	front := httptest.NewServer(secure.Headers(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			// Only the server's own writer can set a deadline.
			err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(time.Minute))
			if err != nil {
				t.Errorf("set the write deadline: %v", err)
			}

			_, _ = io.WriteString(w, "first\n")
			_ = http.NewResponseController(w).Flush()

			<-release

			_, _ = io.WriteString(w, "second\n")
		},
	)))
	t.Cleanup(front.Close)
	// Closing the server waits for the handler, so release it first.
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	res := get(ctx, t, front.URL)
	expect(t, res.Header, headers(appPolicy))

	// The handler waits until the test ends, so the first line arrives by the flush alone.
	line, err := bufio.NewReader(res.Body).ReadString('\n')
	if err != nil || line != "first\n" {
		t.Errorf("read %q, %v; want the first line while the handler waits", line, err)
	}
}

// headers returns the security headers that a response with the policy carries.
func headers(policy string) map[string]string {
	return map[string]string{
		"Content-Security-Policy": policy,
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	}
}

// recorded returns what handler sends behind Headers.
func recorded(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	secure.Headers(handler).ServeHTTP(
		rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
	)

	return rec
}

// proxiedHeader returns the header of the response that a gateway in front of a service with
// handler sends: the service's response through a ReverseProxy and Headers.
func proxiedHeader(t *testing.T, handler http.HandlerFunc) http.Header {
	t.Helper()

	service := httptest.NewServer(handler)
	t.Cleanup(service.Close)

	target, err := url.Parse(service.URL)
	if err != nil {
		t.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) { r.SetURL(target) }}
	front := httptest.NewServer(secure.Headers(proxy))
	t.Cleanup(front.Close)

	return get(t.Context(), t, front.URL).Header
}

// get sends a GET to target and returns its response, whose body closes when the test ends.
func get(ctx context.Context, t *testing.T, target string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = res.Body.Close() })

	return res
}

// expect checks that header has each of the wanted headers once, with its wanted value.
func expect(t *testing.T, header http.Header, want map[string]string) {
	t.Helper()

	for key, value := range want {
		if got := header.Values(key); len(got) != 1 || got[0] != value {
			t.Errorf("%s is %q, want only %q", key, got, value)
		}
	}
}
