package exophase_test

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
)

// player is the profile id of the player whose games the tests list.
const player = "12345"

// now is when the tests list the games, and since is 14 days before it.
var (
	now   = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	since = now.AddDate(0, 0, -14)
)

// reply is a fake Exophase's answer to a request.
type reply struct {
	status int
	body   string
	header http.Header
}

// asked is what a fake Exophase noted of a request.
type asked struct {
	request, userAgent, accept string
}

// fakeExophase is a stand-in Exophase. It answers each request with the reply that pages holds
// for its URI, or with success false, as Exophase answers a page past the last.
type fakeExophase struct {
	pages map[string]reply
	mu    sync.Mutex
	asked []asked
}

// serve starts f on a test server and returns the address of its API.
func (f *fakeExophase) serve(t *testing.T) *url.URL {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		f.asked = append(f.asked, asked{
			r.Method + " " + r.URL.RequestURI(), r.UserAgent(), r.Header.Get("Accept"),
		})

		answer, found := f.pages[r.URL.RequestURI()]
		if !found {
			answer = reply{status: http.StatusOK, body: `{"success":false}`}
		}

		maps.Copy(w.Header(), answer.header)

		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(server.Close)

	api, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return api
}

// notes returns what f noted of the requests it was asked.
func (f *fakeExophase) notes() []asked {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.asked)
}

// requests returns the requests that f was asked, as their methods and URIs.
func (f *fakeExophase) requests() []string {
	var requests []string
	for _, request := range f.notes() {
		requests = append(requests, request.request)
	}

	return requests
}

// pageURI returns the URI of the page of the player's games that number counts from 1.
func pageURI(number int) string {
	return fmt.Sprintf("/public/player/%s/games?page=%d&environment=&sort=1&showHidden=0", player,
		number)
}

// getPages returns the requests for the pages of the player's games that numbers count from 1.
func getPages(numbers ...int) []string {
	var requests []string
	for _, number := range numbers {
		requests = append(requests, "GET "+pageURI(number))
	}

	return requests
}

// ok returns a 200 with body.
func ok(body string) reply {
	return reply{status: http.StatusOK, body: body}
}

// pageOf returns a page of Exophase's games list that lists games.
func pageOf(games ...string) string {
	return `{"success":true,"games":[` + strings.Join(games, ",") + `]}`
}

// listed returns a game named title in Exophase's games list, last played at played, with the
// fields that the client doesn't read too.
func listed(title string, played time.Time) string {
	return fmt.Sprintf(`{"master_id":7,"master_playerid":8,"earned_awards":0,"total_awards":0,`+
		`"percent":0,"lastplayed":%[2]d,"lastplayed_utc":%[2]d,"playtime":"",`+
		`"playtimeUnits":{"hours":0,"minutes":0},"resource_tile":"https://m.example/t.png",`+
		`"meta":{"title":%[1]q,"environment_slug":"psn","platforms":[{"name":"PS5","slug":"ps5"}]}}`,
		title, played.Unix())
}

// gameOf returns the Game of listed's game.
func gameOf(title string, played time.Time) exophase.Game {
	return exophase.Game{Title: title, Platforms: []string{"PS5"}, LastPlayed: played}
}

// list returns what PlayedSince gives when fake is Exophase.
func list(t *testing.T, fake *fakeExophase) ([]exophase.Game, error) {
	t.Helper()

	return exophase.New(fake.serve(t), player).PlayedSince(t.Context(), since)
}

// same reports whether got and want hold the same games in the same order.
func same(got, want []exophase.Game) bool {
	return slices.EqualFunc(got, want, func(a, b exophase.Game) bool {
		return a.Title == b.Title && slices.Equal(a.Platforms, b.Platforms) &&
			a.LastPlayed.Equal(b.LastPlayed) && a.PlaytimeMinutes == b.PlaytimeMinutes &&
			a.Trophies == b.Trophies
	})
}

// padded returns a page of Exophase's games list with no games, size bytes long.
func padded(size int) string {
	start, end := `{"success":true,"games":[],"pad":"`, `"}`

	return start + strings.Repeat("x", size-len(start)-len(end)) + end
}
