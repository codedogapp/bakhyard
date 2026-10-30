package exophase_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
)

// listWith returns what PlayedSince gives when Exophase answers the page that number counts from
// 1 with answer, and the pages before it with a game played since.
func listWith(t *testing.T, number int, answer reply) ([]exophase.Game, error) {
	t.Helper()

	pages := map[string]reply{pageURI(number): answer}
	for before := 1; before < number; before++ {
		pages[pageURI(before)] = ok(pageOf(listed("A", now)))
	}

	return list(t, &fakeExophase{pages: pages})
}

func TestPlayedSinceFailsOnAStatus(t *testing.T) {
	problem := `{"title":"An error occurred","status":500,"detail":"Internal Server Error"}`
	challenge := http.Header{"Cf-Mitigated": {"challenge"}}

	for _, test := range []struct {
		name   string
		answer reply
		want   string
	}{
		{
			"500",
			reply{http.StatusInternalServerError, problem, nil},
			"list my Exophase games: 500 Internal Server Error",
		},
		{
			"429",
			reply{http.StatusTooManyRequests, "", nil},
			"list my Exophase games: 429 Too Many Requests",
		},
		{"403", reply{http.StatusForbidden, "", nil}, "list my Exophase games: 403 Forbidden"},
		{
			"Cloudflare's challenge",
			reply{http.StatusForbidden, "<html>Just a moment</html>", challenge},
			"list my Exophase games: 403 Forbidden, a Cloudflare challenge",
		},
	} {
		for _, number := range []int{1, 2} {
			got, err := listWith(t, number, test.answer)
			if got != nil || err == nil || err.Error() != test.want {
				t.Errorf(
					"%s on page %d: got %v, %v, want the error %q",
					test.name,
					number,
					got,
					err,
					test.want,
				)
			}
		}
	}
}

func TestPlayedSinceFailsOnAnAnswerItCantRead(t *testing.T) {
	for _, test := range []struct{ name, body string }{
		{"not JSON", "<html>Exophase</html>"},
		{"cut short", `{"success":true,"games":[`},
		{"games that aren't a list", `{"success":true,"games":{}}`},
		{"a playtime that isn't an object", `{"success":true,"games":[{"playtimeUnits":[]}]}`},
		{"over 4 MiB", padded(4<<20 + 1)},
	} {
		for _, number := range []int{1, 2} {
			got, err := listWith(t, number, ok(test.body))
			if got != nil || err == nil ||
				!strings.HasPrefix(err.Error(), "read Exophase's answer: ") {
				t.Errorf("%s on page %d: got %v, %v, want an error that starts with %q", test.name,
					number, got, err, "read Exophase's answer: ")
			}
		}
	}
}

func TestPlayedSinceFailsWithoutAPublicProfile(t *testing.T) {
	fake := &fakeExophase{pages: map[string]reply{pageURI(1): ok(`{"success":false}`)}}

	got, err := list(t, fake)

	want := "list my Exophase games: Exophase has no public profile with this id"
	if got != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %q", got, err, want)
	}

	if got, want := fake.requests(), getPages(1); !slices.Equal(got, want) {
		t.Errorf("asked for %q, want %q", got, want)
	}
}

func TestPlayedSinceStopsAfterFourPages(t *testing.T) {
	pages := map[string]reply{}
	for number := 1; number <= 5; number++ {
		pages[pageURI(number)] = ok(pageOf(listed("A", now)))
	}

	fake := &fakeExophase{pages: pages}

	got, err := list(t, fake)

	want := "list my Exophase games: more than 4 pages"
	if got != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %q", got, err, want)
	}

	if got, want := fake.requests(), getPages(1, 2, 3, 4); !slices.Equal(got, want) {
		t.Errorf("asked for %q, want %q", got, want)
	}
}

func TestPlayedSinceFailsWithoutExophase(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	api, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	got, err := exophase.New(api, player).PlayedSince(t.Context(), since)
	if got != nil || err == nil || !strings.HasPrefix(err.Error(), "list my Exophase games: ") {
		t.Errorf("got %v, %v, want an error that starts with %q", got, err,
			"list my Exophase games: ")
	}
}

func TestPlayedSinceFailsOnABadAddress(t *testing.T) {
	api := &url.URL{Scheme: "https", Host: "exophase example"}

	got, err := exophase.New(api, player).PlayedSince(t.Context(), since)
	if got != nil || err == nil || !strings.HasPrefix(err.Error(), "list my Exophase games: ") {
		t.Errorf("got %v, %v, want an error that starts with %q", got, err,
			"list my Exophase games: ")
	}
}

func TestPlayedSinceStopsWhenCancelled(t *testing.T) {
	fake := &fakeExophase{pages: map[string]reply{pageURI(1): ok(pageOf(listed("A", now)))}}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := exophase.New(fake.serve(t), player).PlayedSince(ctx, since)
	if got != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, %v, want %v", got, err, context.Canceled)
	}

	if requests := fake.requests(); requests != nil {
		t.Errorf("asked for %q, want nothing", requests)
	}
}
