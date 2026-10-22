package signin_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/gateway/internal/signin"
)

func TestTheUserSignsIn(t *testing.T) {
	google := newFakeGoogle(t)
	flow, logs := newFlow(t, google.provider())

	redirect := login(t, flow)
	answer := url.Values{
		"state": {redirect.Query().Get("state")},
		"code":  {google.approve(redirect, func(*grant) {})},
	}

	rec := callback(t, flow, answer)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("got %d to %q, want 302 to /", rec.Code, rec.Header().Get("Location"))
	}

	if !strings.Contains(logs.String(), `level=INFO msg="signed in" email=user@example.com`) {
		t.Errorf("got the log %q, want signed in with the email", logs)
	}

	// The state works once.
	refused(t, callback(t, flow, answer), logs, "unknown, used or expired state")
	google.noLeaks(t, logs.String())
}

func TestTheUserSignsInOnSeveralDevicesAtOnce(t *testing.T) {
	google := newFakeGoogle(t)
	flow, logs := newFlow(t, google.provider())

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

	if got := strings.Count(logs.String(), `msg="signed in"`); got != devices {
		t.Errorf("got %d sign-ins in the log, want %d", got, devices)
	}

	google.noLeaks(t, logs.String())
}

func TestTheCallbackRefusesAWrongIDToken(t *testing.T) {
	forger := newKey(t)

	for name, test := range map[string]struct {
		change func(*grant)
		reason string
	}{
		"wrong nonce": {func(g *grant) { g.claims["nonce"] = "other" }, "wrong nonce"},
		"no nonce":    {func(g *grant) { delete(g.claims, "nonce") }, "wrong nonce"},
		"unverified email": {
			func(g *grant) { g.claims["email_verified"] = false },
			"unverified email user@example.com",
		},
		"no email_verified": {
			func(g *grant) { delete(g.claims, "email_verified") },
			"unverified email user@example.com",
		},
		"email_verified as a string": {
			func(g *grant) { g.claims["email_verified"] = "true" },
			"read the ID token's claims",
		},
		"someone else's email": {
			func(g *grant) { g.claims["email"] = "someone@example.com" },
			"someone else's email someone@example.com",
		},
		"the email in other case": {
			func(g *grant) { g.claims["email"] = "User@example.com" },
			"someone else's email User@example.com",
		},
		"another client's token": {
			func(g *grant) { g.claims["aud"] = "other-client" },
			"verify the ID token: oidc: expected audience",
		},
		"another issuer": {
			func(g *grant) { g.claims["iss"] = "https://accounts.example.com" },
			"verify the ID token: oidc: id token issued by a different provider",
		},
		"expired token": {
			func(g *grant) { g.claims["exp"] = time.Now().Add(-time.Minute).Unix() },
			"verify the ID token: oidc: token is expired",
		},
		"forged signature": {
			func(g *grant) { g.signer = forger },
			"verify the ID token: failed to verify signature",
		},
		"no ID token": {func(g *grant) { g.claims = nil }, "no ID token in Google's answer"},
	} {
		t.Run(name, func(t *testing.T) {
			google := newFakeGoogle(t)
			flow, logs := newFlow(t, google.provider())

			redirect := login(t, flow)
			refused(t, callback(t, flow, url.Values{
				"state": {redirect.Query().Get("state")},
				"code":  {google.approve(redirect, test.change)},
			}), logs, test.reason)
			google.noLeaks(t, logs.String())
		})
	}
}

func TestTheCallbackRefusesAWrongCode(t *testing.T) {
	google := newFakeGoogle(t)
	flow, logs := newFlow(t, google.provider())

	// A code for another login fails the PKCE check, as an unknown code does.
	first, second := login(t, flow), login(t, flow)
	for _, code := range []string{google.approve(second, func(*grant) {}), "4/unknown"} {
		refused(t, callback(t, flow, url.Values{
			"state": {first.Query().Get("state")},
			"code":  {code},
		}), logs, `exchange the code: oauth2: \"invalid_grant\"`)

		first = login(t, flow)
	}

	google.noLeaks(t, logs.String())
}

func TestTheCallbackRefusesGooglesError(t *testing.T) {
	flow, logs := newFlow(t, signin.Google())
	answer := url.Values{"state": {login(t, flow).Query().Get("state")}, "error": {"access_denied"}}

	refused(t, callback(t, flow, answer), logs, `google sent the error \"access_denied\"`)
	refused(t, callback(t, flow, answer), logs, "unknown, used or expired state")
}

func TestTheCallbackRefusesAnUnknownState(t *testing.T) {
	flow, logs := newFlow(t, signin.Google())

	for _, state := range []string{"", "made-up"} {
		answer := url.Values{"state": {state}, "code": {"4/code"}}
		refused(t, callback(t, flow, answer), logs, "unknown, used or expired state")
	}
}

func TestTheCallbackRefusesAnExpiredState(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		flow, logs := newFlow(t, signin.Google())
		answer := url.Values{"state": {login(t, flow).Query().Get("state")}, "code": {"4/code"}}

		time.Sleep(10 * time.Minute)
		refused(t, callback(t, flow, answer), logs, "unknown, used or expired state")
	})
}
