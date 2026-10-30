package igdb_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/games/internal/igdb"
)

// found is IGDB's answer with Moon Garden's cover.
const found = `[{"name":"Moon Garden","cover":{"image_id":"co1abc"}}]`

func TestCoverAsksAgainWithANewTokenWhenIGDBRefusesIt(t *testing.T) {
	address, requests := igdbStandIn(t, func(token, _ string) (int, string) {
		if token == "token-1" {
			return http.StatusUnauthorized, `{"message":"Authorization Failure"}`
		}

		return http.StatusOK, found
	})
	given := &tokens{}

	got, err := igdb.New(address, clientID, given).Cover(t.Context(), "Moon Garden")
	if got != covers+"co1abc.jpg" || err != nil {
		t.Errorf("got %q, %v, want the cover", got, err)
	}

	if want := []string{"token-1"}; !slices.Equal(given.dropped, want) {
		t.Errorf("dropped %q, want %q", given.dropped, want)
	}

	want := []string{asked("token-1", searched), asked("token-2", searched)}
	if got := requests(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestCoverAsksOnlyOnceMoreWhenIGDBRefusesTheNewTokenToo(t *testing.T) {
	address, requests := igdbStandIn(t, answerWith(http.StatusUnauthorized))
	given := &tokens{}

	got, err := igdb.New(address, clientID, given).Cover(t.Context(), "Moon Garden")

	want := `find the cover of "Moon Garden": ask IGDB: 401 Unauthorized`
	if got != "" || err == nil || err.Error() != want {
		t.Errorf("got %q, %v, want %q", got, err, want)
	}

	if want := []string{"token-1", "token-2"}; !slices.Equal(given.dropped, want) {
		t.Errorf("dropped %q, want %q", given.dropped, want)
	}

	if got := requests(); len(got) != 2 {
		t.Errorf("got the requests %q, want 2", got)
	}
}

func TestCoverSaysWhyIGDBFailed(t *testing.T) {
	for _, test := range []struct {
		name   string
		answer answer
		want   string
	}{
		{"an error", answerWith(http.StatusInternalServerError), "500 Internal Server Error"},
		{"too many requests", answerWith(http.StatusTooManyRequests), "429 Too Many Requests"},
		{"games found with another status", func(string, string) (int, string) {
			return http.StatusAccepted, found
		}, "202 Accepted"},
		{"the search for the title's name fails", func(_, query string) (int, string) {
			if query == searched {
				return http.StatusOK, `[{"name":"Moon Garden 2"}]`
			}

			return http.StatusServiceUnavailable, ""
		}, "503 Service Unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			address, _ := igdbStandIn(t, test.answer)

			got, err := cover(t, address, "Moon Garden")

			want := `find the cover of "Moon Garden": ask IGDB: ` + test.want
			if got != "" || err == nil || err.Error() != want {
				t.Errorf("got %q, %v, want %q", got, err, want)
			}
		})
	}
}

func TestCoverRefusesAnAnswerItCannotRead(t *testing.T) {
	// 64 KiB of games is the most an answer may hold, padded at its start, so that the cut
	// that would end a longer answer takes its last bracket.
	padded := func(size int) string { return strings.Repeat(" ", size-len(found)) + found }

	for _, body := range []string{
		`<html>`, `{"name":"Moon Garden"}`, `[{"name":1}]`, `[{"cover":"co1abc"}]`, padded(64<<10 + 1),
	} {
		address, _ := igdbStandIn(t, games(body))

		got, err := cover(t, address, "Moon Garden")

		want := `find the cover of "Moon Garden": read IGDB's answer: `
		if got != "" || err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%.20q: got %q, %v, want an error that starts %q", body, got, err, want)
		}
	}

	address, _ := igdbStandIn(t, games(padded(64<<10)))
	if got, err := cover(t, address, "Moon Garden"); got != covers+"co1abc.jpg" || err != nil {
		t.Errorf("64 KiB: got %q, %v, want the cover", got, err)
	}
}

func TestCoverSaysWhyItGotNoToken(t *testing.T) {
	address, requests := igdbStandIn(t, games(found))
	noTokens := &tokens{err: errors.New("get a Twitch token: 400 Bad Request, invalid client")}

	got, err := igdb.New(address, clientID, noTokens).Cover(t.Context(), "Moon Garden")

	want := `find the cover of "Moon Garden": get a Twitch token: 400 Bad Request, invalid client`
	if got != "" || err == nil || err.Error() != want {
		t.Errorf("got %q, %v, want %q", got, err, want)
	}

	if got := requests(); len(got) != 0 {
		t.Errorf("got the requests %q, want none", got)
	}
}

func TestCoverSaysIGDBIsOutOfReach(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	got, err := cover(t, address, "Moon Garden")

	want := `find the cover of "Moon Garden": ask IGDB: Post "` + server.URL + `/v4/games": `
	if got != "" || err == nil || !strings.HasPrefix(err.Error(), want) ||
		strings.Contains(err.Error(), "token-1") {
		t.Errorf("got %q, %v, want an error that starts %q, without the token", got, err, want)
	}
}

// answerWith answers every query with status and no body.
func answerWith(status int) answer {
	return func(string, string) (int, string) { return status, "" }
}
