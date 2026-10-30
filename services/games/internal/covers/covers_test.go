package covers_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/covers"
)

func TestFindAsksForEachTitleOnceInOrder(t *testing.T) {
	finder := finderOf(t, map[string]string{"B": "b.jpg", "A": "a.jpg"})

	find(t, covers.New(finder), []string{"B", "A", "B", "No cover", "A", "No cover"},
		"b.jpg", "a.jpg", "b.jpg", "", "a.jpg", "")
	finder.wantAsked("B", "A", "No cover")
}

func TestFindOfNoTitlesAsksNothing(t *testing.T) {
	finder := finderOf(t, nil)

	find(t, covers.New(finder), nil)
	finder.wantAsked()
}

func TestFindKeepsACoverForAWeek(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		finder := finderOf(t, map[string]string{"A": "a.jpg", "C": "c.jpg"})
		cache := covers.New(finder)

		find(t, cache, []string{"A", "No cover"}, "a.jpg", "")
		time.Sleep(7*24*time.Hour - time.Nanosecond)
		find(t, cache, []string{"A", "No cover", "C"}, "a.jpg", "", "c.jpg")
		time.Sleep(time.Nanosecond)
		find(t, cache, []string{"C", "A", "No cover"}, "c.jpg", "a.jpg", "")

		finder.wantAsked("A", "No cover", "C", "A", "No cover")
	})
}

func TestFindCountsTheWeekFromTheAsking(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		finder := finderOf(t, map[string]string{"A": "a.jpg"})
		finder.delay = time.Minute
		cache := covers.New(finder)

		find(t, cache, []string{"A"}, "a.jpg")
		time.Sleep(7*24*time.Hour - time.Minute)
		find(t, cache, []string{"A"}, "a.jpg")

		finder.wantAsked("A", "A")
	})
}

func TestFindKeepsTheCoversOfTheLatestTitlesOnly(t *testing.T) {
	finder := finderOf(t, map[string]string{"A": "a.jpg", "B": "b.jpg"})
	cache := covers.New(finder)

	find(t, cache, []string{"A", "B"}, "a.jpg", "b.jpg")
	find(t, cache, []string{"A"}, "a.jpg")
	find(t, cache, []string{"B", "A"}, "b.jpg", "a.jpg")

	finder.wantAsked("A", "B", "B")
}

func TestFindAsksNoMoreAfterAFailure(t *testing.T) {
	finder := finderOf(t, map[string]string{
		"Kept": "kept.jpg", "New": "new.jpg", "Down": "down.jpg", "Later": "later.jpg",
	})
	cache := covers.New(finder)

	find(t, cache, []string{"Kept"}, "kept.jpg")

	down := errors.New("IGDB is down")
	finder.fails = map[string]error{"Down": down}

	got, err := cache.Find(t.Context(), []string{"New", "Down", "Kept", "Later", "Down", "New"})

	want := []string{"new.jpg", "", "kept.jpg", "", "", "new.jpg"}
	if !errors.Is(err, down) || !slices.Equal(got, want) {
		t.Errorf("got %q, %v, want %q, %v", got, err, want, down)
	}

	finder.fails = nil

	find(t, cache, []string{"New", "Down", "Kept", "Later"},
		"new.jpg", "down.jpg", "kept.jpg", "later.jpg")

	finder.wantAsked("Kept", "New", "Down", "Down", "Later")
}

// finder is a stand-in Finder, which gives a title's cover from found, with the title's error from
// fails, after delay, and records the titles that it was asked for.
type finder struct {
	t     *testing.T
	found map[string]string
	fails map[string]error
	delay time.Duration
	asked []string
}

// finderOf returns a finder of the covers in found, whose titles may be asked for only with t's
// context.
func finderOf(t *testing.T, found map[string]string) *finder {
	t.Helper()

	return &finder{t: t, found: found}
}

func (f *finder) Cover(ctx context.Context, title string) (string, error) {
	if ctx != f.t.Context() {
		f.t.Errorf("%q: asked with another context", title)
	}

	f.asked = append(f.asked, title)
	time.Sleep(f.delay)

	return f.found[title], f.fails[title]
}

// wantAsked fails the test unless the finder was asked for want, in order.
func (f *finder) wantAsked(want ...string) {
	f.t.Helper()

	if !slices.Equal(f.asked, want) {
		f.t.Errorf("asked for %q, want %q", f.asked, want)
	}
}

// find has cache find the covers of titles, and fails the test unless they're want.
func find(t *testing.T, cache *covers.Cache, titles []string, want ...string) {
	t.Helper()

	got, err := cache.Find(t.Context(), titles)
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("%q: got %q, %v, want %q", titles, got, err, want)
	}
}
