package signin_test

import (
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestTheUserSignsIn(t *testing.T) {
	google := newFakeGoogle(t)
	flow := newFlow(t, google.provider())

	redirect := login(t, flow)
	answer := url.Values{
		"state": {redirect.Query().Get("state")},
		"code":  {google.approve(redirect, func(*grant) {})},
	}

	rec := callback(t, flow, answer)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("got %d to %q, want 302 to /", rec.Code, rec.Header().Get("Location"))
	}

	// One session, whose token only the cookie carries.
	tokens := flow.sessions.started()
	if len(tokens) != 1 {
		t.Fatalf("started %d sessions, want 1", len(tokens))
	}

	want := "__Host-session=" + tokens[0] +
		"; Path=/; Max-Age=604800; HttpOnly; Secure; SameSite=Lax"
	if got := rec.Header().Values("Set-Cookie"); !slices.Equal(got, []string{want}) {
		t.Errorf("got the cookies %q, want %q", got, want)
	}

	logs := flow.logs.String()
	if !strings.Contains(logs, `level=INFO msg="signed in" email=user@example.com`) ||
		strings.Contains(logs, tokens[0]) {
		t.Errorf("got the log %q, want signed in with the email, without the token", logs)
	}

	pinnedTheUser(t, flow)

	// The state works once.
	refused(t, flow, answer, "unknown, used or expired state")
	google.noLeaks(t, flow.logs.String())
}

func TestTheUserSignsInOnSeveralDevicesAtOnce(t *testing.T) {
	google := newFakeGoogle(t)
	flow := newFlow(t, google.provider())

	const devices = 8

	// Run returns once its parallel subtests have finished.
	t.Run("at once", func(t *testing.T) {
		for device := range devices {
			t.Run(strconv.Itoa(device), func(t *testing.T) {
				t.Parallel()

				redirect := login(t, flow)

				rec := callback(t, flow, url.Values{
					"state": {redirect.Query().Get("state")},
					"code":  {google.approve(redirect, func(*grant) {})},
				})
				if rec.Code != http.StatusFound {
					t.Errorf("got %d, want 302", rec.Code)
				}
			})
		}
	})

	signedIn, started := strings.Count(flow.logs.String(), `msg="signed in"`),
		len(flow.sessions.started())
	if signedIn != devices || started != devices {
		t.Errorf("got %d sign-ins in the log and %d sessions, want %d of each",
			signedIn, started, devices)
	}

	pinnedTheUser(t, flow)
	google.noLeaks(t, flow.logs.String())
}

func TestTheCallbackFailsWhenAStoreFails(t *testing.T) {
	for name, fail := range map[string]func(*testFlow, error){
		"the pin store":     func(flow *testFlow, err error) { flow.pins.err = err },
		"the session store": func(flow *testFlow, err error) { flow.sessions.err = err },
	} {
		t.Run(name, func(t *testing.T) {
			google := newFakeGoogle(t)
			flow := newFlow(t, google.provider())
			fail(flow, errors.New("the database is down"))

			redirect := login(t, flow)
			rec := callback(t, flow, url.Values{
				"state": {redirect.Query().Get("state")},
				"code":  {google.approve(redirect, func(*grant) {})},
			})

			cookies, started := rec.Header().Values("Set-Cookie"), len(flow.sessions.started())
			if rec.Code != http.StatusInternalServerError || len(cookies) != 0 || started != 0 {
				t.Errorf("got %d with the cookies %q and %d sessions, want 500 without any",
					rec.Code, cookies, started)
			}

			line := `level=ERROR msg="failed a sign-in" email=user@example.com ` +
				`err="the database is down"`
			if !strings.Contains(flow.logs.String(), line) {
				t.Errorf("got the log %q, want %s", flow.logs, line)
			}

			google.noLeaks(t, flow.logs.String())
		})
	}
}

// pinnedTheUser fails t unless flow has pinned the user's account to their email, and nothing
// else.
func pinnedTheUser(t *testing.T, flow *testFlow) {
	t.Helper()

	want := map[string]string{email: subject}
	if got := flow.pins.pinned(); !maps.Equal(got, want) {
		t.Errorf("got the pins %v, want %v", got, want)
	}
}
