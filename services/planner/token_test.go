package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/githubapi"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

func TestPlannerRemindsToRenewTheCopilotTokenBeforeItExpires(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", cliOn(t, executable))

	expires := time.Now().Add(3 * 24 * time.Hour).Truncate(time.Second)
	getenv := copilotEnv(t.TempDir(), githubServer(t, expires))
	ctx, stop := context.WithCancel(t.Context())

	var stopping sync.WaitGroup

	// The CLI stops before the test's folders go.
	defer stopping.Wait()
	defer stop()

	router, err := routes(ctx, getenv, secretsWith(token), &stopping)
	if err != nil {
		t.Fatal(err)
	}

	got := listItems(t, router)

	want := "copilot-token | Renew the Copilot token |  | SOURCE_COPILOT | " +
		expires.UTC().String()
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %q, want %q alone", got, want)
	}
}

func TestPlannerListsNoItemsWithoutCopilot(t *testing.T) {
	var stopping sync.WaitGroup

	router, err := routes(t.Context(), envOf(nil), secretsWith(nil), &stopping)
	if err != nil {
		t.Fatal(err)
	}

	if got := listItems(t, router); len(got) != 0 {
		t.Errorf("got %q, want no items", got)
	}
}

func TestPlannerStopsOnABadCopilotAPIBeforeTheCLIStarts(t *testing.T) {
	// A CLI that fails would fail the start, had the API not stopped planner first.
	t.Setenv("PATH", cliOn(t, "/usr/bin/false"))

	var stopping sync.WaitGroup

	_, err := routes(t.Context(), copilotEnv(t.TempDir(), "http://example.com"),
		secretsWith(token), &stopping)

	want := `BAKHYARD_COPILOT_API: "http://example.com" is not an https URL`
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}

func TestCopilotAPIComesFromTheEnvironment(t *testing.T) {
	copilotOn := copilot.Settings{Host: "example.ghe.com", Token: "secret"}

	for _, test := range []struct {
		name, api string
		settings  copilot.Settings
		want      string
	}{
		{"the host's", "", copilotOn, "https://api.example.ghe.com"},
		{"a server's", "https://github.example.com/api/v3", copilotOn, "https://github.example.com/api/v3"},
		{"a stand-in's", "http://127.0.0.1:8785", copilotOn, "http://127.0.0.1:8785"},
		{"none without Copilot", "http://example.com", copilot.Settings{}, ""},
	} {
		got, err := copilotAPI(envOf(map[string]string{apiVar: test.api}), test.settings)

		address := ""
		if got.API != nil {
			address = got.API.String()
		}

		if err != nil || address != test.want || got.Token != test.settings.Token {
			t.Errorf("%s: got %q with the token %q, and %v, want %q with %q", test.name, address,
				got.Token, err, test.want, test.settings.Token)
		}
	}
}

func TestCopilotAPIMustBeSecure(t *testing.T) {
	for _, value := range []string{"http://example.com", "api.example.ghe.com", "https://"} {
		got, err := copilotAPI(envOf(map[string]string{apiVar: value}),
			copilot.Settings{Host: "example.ghe.com", Token: "secret"})

		want := fmt.Sprintf("BAKHYARD_COPILOT_API: %q is not an https URL", value)
		if err == nil || err.Error() != want || got != (githubapi.Settings{}) {
			t.Errorf("%q: got %+v and %v, want the error %s", value, got, err, want)
		}
	}
}

func TestCopilotTokenExpiresWhenGitHubSays(t *testing.T) {
	expires := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)

	api, err := url.Parse(githubServer(t, expires))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		settings githubapi.Settings
		want     time.Time
		logged   string
	}{
		{"GitHub's answer", githubapi.Settings{API: api, Token: "secret"}, expires, ""},
		{
			"a failed ask",
			githubapi.Settings{API: api, Token: "wrong"},
			time.Time{},
			`level=ERROR msg="poll Copilot token" ` +
				`err="ask GitHub when the token expires: 401 Unauthorized"`,
		},
		{"Copilot off", githubapi.Settings{}, time.Time{}, ""},
	} {
		var logs bytes.Buffer

		token := copilotToken(t.Context(), test.settings, slog.New(slog.NewTextHandler(&logs, nil)))

		got, err := token.Expires(t.Context())
		if err != nil || !got.Equal(test.want) || token.Name != "Copilot" ||
			token.Source != attentionv1.Source_SOURCE_COPILOT {
			t.Errorf("%s: got %+v, expiring %v, and %v, want the Copilot token expiring %v",
				test.name, token, got, err, test.want)
		}

		if line := strings.TrimSuffix(logs.String(), "\n"); !strings.HasSuffix(line, test.logged) ||
			(test.logged == "") != (line == "") {
			t.Errorf("%s: logged %q, want %q", test.name, line, test.logged)
		}
	}
}

// githubServer returns the address of a stand-in for GitHub's API, which tells the token secret
// that it expires at expires, or never when expires is zero, and turns any other token away.
func githubServer(t *testing.T, expires time.Time) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" || r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if !expires.IsZero() {
			w.Header().Set("GitHub-Authentication-Token-Expiration",
				expires.UTC().Format(time.DateTime+" UTC"))
		}

		_, _ = w.Write([]byte(`{"login":"you"}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// listItems returns the items that router lists over the attention contract, one line each: their
// ID, title, detail, source and due time.
func listItems(t *testing.T, router http.Handler) []string {
	t.Helper()

	// gRPC needs HTTP/2; between services it runs without TLS (h2c), as internal/server allows.
	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(router)
	server.Config.Protocols = h2c
	server.Start()
	t.Cleanup(server.Close)

	transport := &http.Transport{Protocols: h2c}
	t.Cleanup(transport.CloseIdleConnections)

	client := attentionv1connect.NewAttentionServiceClient(connect.NewClient(
		connecthttp.NewTransport(&http.Client{Transport: transport}, server.URL,
			connecthttp.WithGRPC())))

	res, err := client.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var lines []string

	for _, item := range res.GetItems() {
		due := "no due"
		if item.HasDue() {
			due = item.GetDue().AsTime().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", item.GetId(),
			item.GetTitle(), item.GetDetail(), item.GetSource(), due))
	}

	return lines
}
