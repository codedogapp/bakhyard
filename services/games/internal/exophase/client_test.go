package exophase_test

import (
	"slices"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
)

func TestPlayedSinceReadsThePagesItNeeds(t *testing.T) {
	hourAgo, twoHoursAgo, threeHoursAgo := now.Add(-time.Hour), now.Add(-2*time.Hour),
		now.Add(-3*time.Hour)
	old := since.Add(-time.Second)

	for _, test := range []struct {
		name  string
		pages map[string]reply
		want  []exophase.Game
		// asked are the pages that the client asks for, counted from 1.
		asked []int
	}{
		{
			name:  "a page that ends before since",
			pages: map[string]reply{pageURI(1): ok(pageOf(listed("A", hourAgo), listed("Z", old)))},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1},
		},
		{
			name:  "a game played at since",
			pages: map[string]reply{pageURI(1): ok(pageOf(listed("A", since), listed("Z", old)))},
			want:  []exophase.Game{gameOf("A", since)},
			asked: []int{1},
		},
		{
			name:  "no game since",
			pages: map[string]reply{pageURI(1): ok(pageOf(listed("Z", old)))},
			asked: []int{1},
		},
		{
			name: "two pages",
			pages: map[string]reply{
				pageURI(1): ok(pageOf(listed("A", hourAgo), listed("B", twoHoursAgo))),
				pageURI(2): ok(pageOf(listed("C", threeHoursAgo), listed("Z", old))),
			},
			want:  []exophase.Game{gameOf("A", hourAgo), gameOf("B", twoHoursAgo), gameOf("C", threeHoursAgo)},
			asked: []int{1, 2},
		},
		{
			name:  "a page past the last",
			pages: map[string]reply{pageURI(1): ok(pageOf(listed("A", hourAgo)))},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1, 2},
		},
		{
			name: "an empty page",
			pages: map[string]reply{
				pageURI(1): ok(pageOf(listed("A", hourAgo))), pageURI(2): ok(pageOf()),
			},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1, 2},
		},
		{
			name: "a page past the last that lists games",
			pages: map[string]reply{
				pageURI(1): ok(pageOf(listed("A", hourAgo))),
				pageURI(2): ok(`{"success":false,"games":[` + listed("B", twoHoursAgo) + "]}"),
			},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1, 2},
		},
		{
			name: "a page whose games are null",
			pages: map[string]reply{
				pageURI(1): ok(pageOf(listed("A", hourAgo))),
				pageURI(2): ok(`{"success":true,"games":null}`),
			},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1, 2},
		},
		{
			name: "a page of 4 MiB",
			pages: map[string]reply{
				pageURI(1): ok(pageOf(listed("A", hourAgo))), pageURI(2): ok(padded(4 << 20)),
			},
			want:  []exophase.Game{gameOf("A", hourAgo)},
			asked: []int{1, 2},
		},
		{
			name: "games out of order",
			pages: map[string]reply{pageURI(1): ok(pageOf(
				listed("B", twoHoursAgo), listed("A", hourAgo), listed("Z", old), listed("C", threeHoursAgo),
			))},
			want:  []exophase.Game{gameOf("A", hourAgo), gameOf("B", twoHoursAgo), gameOf("C", threeHoursAgo)},
			asked: []int{1, 2},
		},
		{
			name: "games last played at the same time",
			pages: map[string]reply{pageURI(1): ok(pageOf(
				listed("B", hourAgo), listed("A", hourAgo), listed("Z", old),
			))},
			want:  []exophase.Game{gameOf("B", hourAgo), gameOf("A", hourAgo)},
			asked: []int{1},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeExophase{pages: test.pages}

			got, err := list(t, fake)
			if err != nil || !same(got, test.want) {
				t.Errorf("got %v, %v, want %v", got, err, test.want)
			}

			if got, want := fake.requests(), getPages(test.asked...); !slices.Equal(got, want) {
				t.Errorf("asked for %q, want %q", got, want)
			}
		})
	}
}

func TestPlayedSinceAsksAsBakhyard(t *testing.T) {
	fake := &fakeExophase{pages: map[string]reply{
		pageURI(1): ok(pageOf(listed("Z", since.Add(-time.Second)))),
	}}

	_, err := list(t, fake)
	if err != nil {
		t.Fatal(err)
	}

	want := []asked{{
		"GET " + pageURI(1), "bakhyard/0.1 (+https://github.com/codedogapp/bakhyard)",
		"application/json",
	}}
	if got := fake.notes(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestPlayedSinceKeepsThePlayerInOneSegment(t *testing.T) {
	fake := &fakeExophase{}

	_, _ = exophase.New(fake.serve(t), "12/34?").PlayedSince(t.Context(), since)

	want := []string{"GET /public/player/12%2F34%3F/games?page=1&environment=&sort=1&showHidden=0"}
	if got := fake.requests(); !slices.Equal(got, want) {
		t.Errorf("asked for %q, want %q", got, want)
	}
}
