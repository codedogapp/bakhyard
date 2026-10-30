package main

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestGamesListsTheGamesIPlayedInTheLast14Days(t *testing.T) {
	address, asked := exophaseStandIn(t, time.Now())

	router, err := routes(t.Context(), envOf(map[string]string{exophasePlayerVar: "123"}),
		secretsWith(nil), exophaseAt(address))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/games/recent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var answer struct {
		Games []struct {
			Title string `json:"title"`
		} `json:"games"`
		Searched time.Time `json:"searched"`
		Failure  string    `json:"failure"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatalf("got %d %s: %v", rec.Code, rec.Body, err)
	}

	titles := make([]string, 0, len(answer.Games))
	for _, game := range answer.Games {
		titles = append(titles, game.Title)
	}

	want := []string{"An hour ago", "13 days and 23 hours ago"}
	if !slices.Equal(titles, want) || answer.Searched.IsZero() || answer.Failure != "" {
		t.Errorf("got %s, want the games %q", rec.Body, want)
	}

	if got := asked(); !slices.Equal(got, []string{"/public/player/123/games"}) {
		t.Errorf("got the requests %q, want one for the first page of player 123", got)
	}
}

func TestGamesTellsWhyExophaseFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	router, err := routes(t.Context(), envOf(map[string]string{exophasePlayerVar: "123"}),
		secretsWith(nil), exophaseAt(address))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/games/recent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	want := `{"games":[],"failure":"list my Exophase games: 500 Internal Server Error"}`
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("got %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

func TestExophaseIsOffWithoutThePlayer(t *testing.T) {
	client, err := exophaseClient(envOf(nil), unasked())
	if client != nil || err != nil {
		t.Errorf("got %v, %v, want no client and no error", client, err)
	}
}

func TestExophaseClientTakesAPlayerOfDigits(t *testing.T) {
	client, err := exophaseClient(envOf(map[string]string{exophasePlayerVar: "9876543210"}),
		unasked())
	if client == nil || err != nil {
		t.Errorf("got %v, %v, want a client and no error", client, err)
	}
}

func TestExophaseClientRejectsAPlayerThatIsNotDigits(t *testing.T) {
	for _, player := range []string{
		"me", "12a", "-12", "+12", "1.5", "12/", "12:", " 12", "12\n", "1 2", "١٢",
	} {
		client, err := exophaseClient(envOf(map[string]string{exophasePlayerVar: player}),
			unasked())

		want := fmt.Sprintf("BAKHYARD_EXOPHASE_PLAYER: %q is not an Exophase profile id, "+
			"which is digits only", player)
		if client != nil || err == nil || err.Error() != want {
			t.Errorf("%q: got %v, %v, want no client and the error %q", player, client, err, want)
		}
	}
}

// exophaseStandIn starts a stand-in Exophase whose games list has a game played an hour before
// now, one played 13 days and 23 hours before, and one played 14 days and an hour before, and
// returns its address and a function that returns the paths it was asked for.
func exophaseStandIn(t *testing.T, now time.Time) (*url.URL, func() []string) {
	t.Helper()

	var (
		lock  sync.Mutex
		paths []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		paths = append(paths, r.URL.Path)

		_, _ = fmt.Fprintf(w, `{"success":true,"games":[%s,%s,%s]}`,
			listed("An hour ago", now.Add(-time.Hour)),
			listed("13 days and 23 hours ago", now.Add(-13*24*time.Hour-23*time.Hour)),
			listed("14 days and an hour ago", now.Add(-14*24*time.Hour-time.Hour)))
	}))
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return address, func() []string {
		lock.Lock()
		defer lock.Unlock()

		return slices.Clone(paths)
	}
}

// listed returns a game in Exophase's games list, with title, last played at played.
func listed(title string, played time.Time) string {
	return fmt.Sprintf(`{"lastplayed_utc":%d,"meta":{"title":%q,"platforms":[]}}`,
		played.Unix(), title)
}
