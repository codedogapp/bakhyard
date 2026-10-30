package twitch_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/games/internal/twitch"
)

// secret is the stand-in app's client secret, which no error may show.
const secret = "hijklmn67890"

func TestTokensAskTwitchForAnAppToken(t *testing.T) {
	address, asked := twitchStandIn(
		t,
		http.StatusOK,
		`{"access_token":"an-app-token","expires_in":5011271,"token_type":"bearer"}`,
	)
	tokens := twitch.New(address, "abcdefg12345", secret)

	for range 2 {
		got, err := tokens.Token(t.Context())
		if got != "an-app-token" || err != nil {
			t.Fatalf("got %q, %v, want the token", got, err)
		}
	}

	want := []string{"POST /oauth2/token?, Content-Type application/x-www-form-urlencoded, " +
		"Accept application/json: " +
		"client_id=abcdefg12345&client_secret=hijklmn67890&grant_type=client_credentials"}
	if got := asked(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestTokensSayWhyTwitchRefused(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
	}{
		{
			"no such client", `{"status":400,"message":"invalid client"}`,
			"get a Twitch token: 400 Bad Request, invalid client", http.StatusBadRequest,
		},
		{
			"a wrong secret", `{"status":403,"message":"invalid client secret"}`,
			"get a Twitch token: 403 Forbidden, invalid client secret", http.StatusForbidden,
		},
		{
			"no reason", `{"status":502}`,
			"get a Twitch token: 502 Bad Gateway", http.StatusBadGateway,
		},
		{
			"not JSON", "<p>Down</p>",
			"get a Twitch token: 500 Internal Server Error", http.StatusInternalServerError,
		},
		{
			"a reason over 64 KiB", `{"message":"` + strings.Repeat("a", 64<<10) + `"}`,
			"get a Twitch token: 500 Internal Server Error", http.StatusInternalServerError,
		},
		{
			"a status that isn't 200", `{"access_token":"a","expires_in":100}`,
			"get a Twitch token: 202 Accepted", http.StatusAccepted,
		},
		{"no token", `{"expires_in":100}`, "read Twitch's answer: no access token", http.StatusOK},
		{
			"no lifetime", `{"access_token":"a"}`,
			"read Twitch's answer: the token expires in 0 seconds", http.StatusOK,
		},
		{
			"a lifetime past", `{"access_token":"a","expires_in":-1}`,
			"read Twitch's answer: the token expires in -1 seconds", http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			address, _ := twitchStandIn(t, test.status, test.body)

			token, err := twitch.New(address, "abcdefg12345", secret).Token(t.Context())
			if token != "" || err == nil || err.Error() != test.want {
				t.Errorf("got %q, %v, want the error %q", token, err, test.want)
			}
		})
	}
}

func TestTokensRefuseAnAnswerTheyCannotRead(t *testing.T) {
	for _, body := range []string{
		"<p>Hi</p>",
		`{"access_token":"` + strings.Repeat("a", 64<<10) + `","expires_in":100}`,
	} {
		address, _ := twitchStandIn(t, http.StatusOK, body)

		token, err := twitch.New(address, "abcdefg12345", secret).Token(t.Context())
		if token != "" || err == nil || !strings.HasPrefix(err.Error(), "read Twitch's answer: ") {
			t.Errorf("%.20s: got %q, %v, want an error that starts %q", body, token, err,
				"read Twitch's answer: ")
		}
	}
}

func TestTokensSayTwitchIsOutOfReach(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	token, err := twitch.New(address, "abcdefg12345", secret).Token(t.Context())

	want := `get a Twitch token: Post "` + server.URL + `/oauth2/token": `
	if token != "" || err == nil || !strings.HasPrefix(err.Error(), want) ||
		strings.Contains(err.Error(), secret) {
		t.Errorf("got %q, %v, want an error that starts %q, without the secret", token, err, want)
	}
}

// twitchStandIn starts a stand-in Twitch that answers each request with status and body, and
// returns its address and a function that returns the requests it got: each one's method, path,
// query, content type, accepted type and form.
func twitchStandIn(t *testing.T, status int, body string) (*url.URL, func() []string) {
	t.Helper()

	var (
		lock     sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		err := r.ParseForm()
		if err != nil {
			t.Error(err)
		}

		requests = append(requests, fmt.Sprintf("%s %s?%s, Content-Type %s, Accept %s: %s",
			r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"),
			r.Header.Get("Accept"), r.PostForm.Encode()))

		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return address, func() []string {
		lock.Lock()
		defer lock.Unlock()

		return slices.Clone(requests)
	}
}
