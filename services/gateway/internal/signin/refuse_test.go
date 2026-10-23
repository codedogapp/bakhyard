package signin_test

import (
	"net/http"
	"net/url"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/gateway/internal/signin"
)

func TestTheCallbackRefusesAnotherAccountWithTheEmail(t *testing.T) {
	google := newFakeGoogle(t)
	flow := newFlow(t, google.provider())

	redirect := login(t, flow)

	rec := callback(t, flow, url.Values{
		"state": {redirect.Query().Get("state")},
		"code":  {google.approve(redirect, func(*grant) {})},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("the user's sign-in: got %d, want 302", rec.Code)
	}

	redirect = login(t, flow)
	refused(t, flow, url.Values{
		"state": {redirect.Query().Get("state")},
		"code":  {google.approve(redirect, func(g *grant) { g.claims["sub"] = "another" })},
	}, "another account with the email user@example.com")
	pinnedTheUser(t, flow)
	google.noLeaks(t, flow.logs.String())
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
		"no sub":       {func(g *grant) { delete(g.claims, "sub") }, "no sub in the ID token"},
		"an empty sub": {func(g *grant) { g.claims["sub"] = "" }, "no sub in the ID token"},
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
			flow := newFlow(t, google.provider())

			redirect := login(t, flow)
			refused(t, flow, url.Values{
				"state": {redirect.Query().Get("state")},
				"code":  {google.approve(redirect, test.change)},
			}, test.reason)
			google.noLeaks(t, flow.logs.String())
		})
	}
}

func TestTheCallbackRefusesAWrongCode(t *testing.T) {
	google := newFakeGoogle(t)
	flow := newFlow(t, google.provider())

	// A code for another login fails the PKCE check, as an unknown code does.
	first, second := login(t, flow), login(t, flow)
	for _, code := range []string{google.approve(second, func(*grant) {}), "4/unknown"} {
		refused(t, flow, url.Values{
			"state": {first.Query().Get("state")},
			"code":  {code},
		}, `exchange the code: oauth2: \"invalid_grant\"`)

		first = login(t, flow)
	}

	google.noLeaks(t, flow.logs.String())
}

func TestTheCallbackRefusesGooglesError(t *testing.T) {
	flow := newFlow(t, signin.Google())
	answer := url.Values{"state": {login(t, flow).Query().Get("state")}, "error": {"access_denied"}}

	refused(t, flow, answer, `google sent the error \"access_denied\"`)
	refused(t, flow, answer, "unknown, used or expired state")
}

func TestTheCallbackRefusesAnUnknownState(t *testing.T) {
	flow := newFlow(t, signin.Google())

	for _, state := range []string{"", "made-up"} {
		answer := url.Values{"state": {state}, "code": {"4/code"}}
		refused(t, flow, answer, "unknown, used or expired state")
	}
}

func TestTheCallbackRefusesAnExpiredState(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		flow := newFlow(t, signin.Google())
		answer := url.Values{"state": {login(t, flow).Query().Get("state")}, "code": {"4/code"}}

		time.Sleep(10 * time.Minute)
		refused(t, flow, answer, "unknown, used or expired state")
	})
}
