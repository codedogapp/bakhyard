package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/api"
	"github.com/codedogapp/bakhyard/services/games/internal/covers"
	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
)

func TestGamesComeWithTheirCovers(t *testing.T) {
	addresses, asked := coversStandIn(t, map[string]string{
		"An hour ago": "co1", "13 days and 23 hours ago": "co2",
	})
	addresses.exophase, _ = exophaseStandIn(t, time.Now())

	router, err := routes(t.Context(),
		envOf(map[string]string{exophasePlayerVar: "123", twitchClientIDVar: "app"}),
		secretsWith(map[string]string{twitchAccount: "s3cret"}),
		addresses)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/games/recent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var answer struct {
		Games []struct {
			Title string `json:"title"`
			Cover string `json:"cover"`
		} `json:"games"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatalf("got %d %s: %v", rec.Code, rec.Body, err)
	}

	got := make([]string, 0, len(answer.Games))
	for _, game := range answer.Games {
		got = append(got, game.Title+": "+game.Cover)
	}

	want := []string{
		"An hour ago: https://images.igdb.com/igdb/image/upload/t_cover_big/co1.jpg",
		"13 days and 23 hours ago: https://images.igdb.com/igdb/image/upload/t_cover_big/co2.jpg",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %s, want the games and covers %q", rec.Body, want)
	}

	wantAsked := []string{
		"Twitch: app s3cret",
		"IGDB: app, Bearer token",
		"IGDB: app, Bearer token",
	}
	if got := asked(); !slices.Equal(got, wantAsked) {
		t.Errorf("got the requests %q, want %q", got, wantAsked)
	}
}

func TestGamesDoesNotStartWithoutTheTwitchSecret(t *testing.T) {
	router, err := routes(t.Context(), envOf(map[string]string{twitchClientIDVar: "app"}),
		secretsWith(nil), exophaseAt(unasked()))

	want := "read the Twitch client secret: no secret twitch-client-secret"
	if router != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want no handler and the error %q", router, err, want)
	}
}

func TestCoversAreOffWithoutTheTwitchApp(t *testing.T) {
	cache, err := coverCache(t.Context(), envOf(nil), secretsWith(nil), exophaseAt(unasked()))
	if cache != nil || err != nil {
		t.Errorf("got %v, %v, want no cache and no error", cache, err)
	}
}

func TestCoveredGivesNoCoversWithoutTheCache(t *testing.T) {
	games := []exophase.Game{{Title: "A", Platforms: []string{"PS5"}}, {Title: "B"}}

	got := covered(t.Context(), games, nil, slog.New(slog.DiscardHandler))

	want := []api.Game{{Game: games[0]}, {Game: games[1]}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCoveredLogsWhyItFoundNoMoreCovers(t *testing.T) {
	cache := covers.New(finder(func(_ context.Context, title string) (string, error) {
		if title == "B" {
			return "", errors.New("IGDB is down")
		}

		return title + ".jpg", nil
	}))
	games := []exophase.Game{{Title: "A"}, {Title: "B"}, {Title: "C"}}

	var log bytes.Buffer

	got := covered(t.Context(), games, cache, slog.New(slog.NewTextHandler(&log, nil)))

	want := []api.Game{{Game: games[0], Cover: "A.jpg"}, {Game: games[1]}, {Game: games[2]}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if wantLog := `level=ERROR msg="find covers" err="IGDB is down"`; !strings.Contains(
		log.String(), wantLog) {
		t.Errorf("logged %q, want %q", log.String(), wantLog)
	}
}

func TestCoveredLogsNothingWithoutAFailureToTell(t *testing.T) {
	tests := map[string]struct {
		ended bool
		want  []api.Game
	}{
		"every cover found":     {want: []api.Game{{Title: "A", Cover: "A.jpg"}}},
		"the context has ended": {ended: true, want: []api.Game{{Title: "A"}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if test.ended {
				cancel()
			}

			cache := covers.New(finder(func(ctx context.Context, title string) (string, error) {
				return title + ".jpg", ctx.Err()
			}))

			var log bytes.Buffer

			got := covered(
				ctx,
				[]exophase.Game{{Title: "A"}},
				cache,
				slog.New(slog.NewTextHandler(&log, nil)),
			)

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}

			if log.Len() != 0 {
				t.Errorf("logged %q, want nothing", log.String())
			}
		})
	}
}

// finder finds covers as the function does.
type finder func(ctx context.Context, title string) (string, error)

func (f finder) Cover(ctx context.Context, title string) (string, error) {
	return f(ctx, title)
}

// coversStandIn starts stand-ins for Twitch's OAuth server, which gives the token "token", and
// for IGDB, whose game of each title in images is so named and has the cover with the image id
// that images maps the title to. It returns their addresses, with no Exophase, and a function that
// returns the client credentials that each request came with, in order.
func coversStandIn(t *testing.T, images map[string]string) (apis, func() []string) {
	t.Helper()

	var (
		lock     sync.Mutex
		requests []string
	)

	record := func(request string) {
		lock.Lock()
		defer lock.Unlock()

		requests = append(requests, request)
	}

	twitchAPI := http.NewServeMux()
	twitchAPI.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		record("Twitch: " + r.PostFormValue("client_id") + " " + r.PostFormValue("client_secret"))

		_, _ = io.WriteString(w, `{"access_token":"token","expires_in":5000000}`)
	})

	igdbAPI := http.NewServeMux()
	igdbAPI.HandleFunc("POST /v4/games", func(w http.ResponseWriter, r *http.Request) {
		record("IGDB: " + r.Header.Get("Client-ID") + ", " + r.Header.Get("Authorization"))

		query, _ := io.ReadAll(r.Body)
		_, quoted, _ := strings.Cut(string(query), `"`)
		title, _, _ := strings.Cut(quoted, `"`)

		_ = json.MarshalWrite(w, []map[string]any{
			{"name": title, "cover": map[string]string{"image_id": images[title]}},
		})
	})

	return apis{twitch: serve(t, twitchAPI), igdb: serve(t, igdbAPI)}, func() []string {
		lock.Lock()
		defer lock.Unlock()

		return slices.Clone(requests)
	}
}

// serve starts a server of handler and returns its address.
func serve(t *testing.T, handler http.Handler) *url.URL {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return address
}
