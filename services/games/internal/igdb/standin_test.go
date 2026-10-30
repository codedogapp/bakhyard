package igdb_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/games/internal/igdb"
)

// clientID is the stand-in Twitch app's client ID.
const clientID = "abcdefg12345"

// answer gives the status and body of the stand-in IGDB's answer to query, asked with token.
type answer func(token, query string) (int, string)

// games answers every query with 200 and body.
func games(body string) answer {
	return func(string, string) (int, string) { return http.StatusOK, body }
}

// tokens stands in for Twitch's tokens: it gives token-1 until that's dropped, then token-2, and so
// on, or fails with err when it's set.
type tokens struct {
	dropped []string
	err     error
}

func (t *tokens) Token(context.Context) (string, error) {
	if t.err != nil {
		return "", t.err
	}

	return fmt.Sprintf("token-%d", len(t.dropped)+1), nil
}

func (t *tokens) Drop(token string) {
	t.dropped = append(t.dropped, token)
}

// igdbStandIn starts a stand-in IGDB that answers with answer, and returns its address and a
// function that returns the requests it got, as asked writes them.
func igdbStandIn(t *testing.T, answer answer) (*url.URL, func() []string) {
	t.Helper()

	var (
		lock     sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		query, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}

		requests = append(requests, fmt.Sprintf(
			"%s %s, Client-ID %s, Authorization %s, Accept %s, Content-Type %s: %s",
			r.Method, r.URL.Path, r.Header.Get("Client-ID"), r.Header.Get("Authorization"),
			r.Header.Get("Accept"), r.Header.Get("Content-Type"), query))

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		status, body := answer(token, string(query))
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

// asked returns the record of a request for query with token, as the stand-in IGDB writes it.
func asked(token, query string) string {
	return "POST /v4/games, Client-ID " + clientID + ", Authorization Bearer " + token +
		", Accept application/json, Content-Type text/plain; charset=utf-8: " + query
}

// cover returns the cover that a Client finds for title at the stand-in IGDB at address.
func cover(t *testing.T, address *url.URL, title string) (string, error) {
	t.Helper()

	return igdb.New(address, clientID, &tokens{}).Cover(t.Context(), title)
}
