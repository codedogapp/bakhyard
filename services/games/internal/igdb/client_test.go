package igdb_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/services/games/internal/igdb"
)

const (
	// searched is the filtered search for Moon Garden.
	searched = `search "Moon Garden"; fields name,cover.image_id; ` +
		`where game_type = (0,2,4,8,9,10,11) & version_parent = null; limit 1;`
	// named is the search for a game named Moon Garden.
	named = `fields name,cover.image_id; where name ~ "Moon Garden"; limit 1;`
	// covers is where IGDB keeps the covers' JPEGs.
	covers = "https://images.igdb.com/igdb/image/upload/t_cover_big/"
)

func TestCoverAsksIGDBForTheTitle(t *testing.T) {
	address, requests := igdbStandIn(t,
		games(`[{"id":1,"name":"Moon Garden","cover":{"id":2,"image_id":"co1abc"}}]`))

	got, err := cover(t, address, "Moon Garden")
	if got != covers+"co1abc.jpg" || err != nil {
		t.Errorf("got %q, %v, want the cover", got, err)
	}

	want := []string{asked("token-1", searched)}
	if got := requests(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestCoverFallsBackToAGameNamedTheTitle(t *testing.T) {
	one := []string{asked("token-1", searched)}
	both := []string{asked("token-1", searched), asked("token-1", named)}

	for _, test := range []struct {
		name, top, same, want string
		requests              []string
	}{
		{
			"the top result has the title in another case",
			`[{"name":"MOON garden","cover":{"image_id":"top"}}]`, `[]`, "top", one,
		},
		{"the top result has the title and no cover", `[{"name":"Moon Garden"}]`, `[]`, "", one},
		{"the top result has a null cover", `[{"name":"Moon Garden","cover":null}]`, `[]`, "", one},
		{
			"a game has the title", `[{"name":"Moon Garden 2","cover":{"image_id":"top"}}]`,
			`[{"name":"Moon Garden","cover":{"image_id":"same"}}]`, "same", both,
		},
		{
			"the game that has the title has no cover",
			`[{"name":"Moon Garden 2","cover":{"image_id":"top"}}]`, `[{"name":"Moon Garden"}]`,
			"", both,
		},
		{
			"no game has the title",
			`[{"name":"Moon Garden: Remastered","cover":{"image_id":"top"}}]`, `[]`, "top", both,
		},
		{
			"no top result", `[]`, `[{"name":"moon garden","cover":{"image_id":"same"}}]`, "same",
			both,
		},
		{"no game at all", `[]`, `[]`, "", both},
		{"null answers", `null`, `null`, "", both},
		{
			"a cover whose id has a slash and a space",
			`[{"name":"Moon Garden","cover":{"image_id":"a/b c"}}]`, `[]`, "a%2Fb%20c", one,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			address, requests := igdbStandIn(t, func(_, query string) (int, string) {
				if query == searched {
					return http.StatusOK, test.top
				}

				return http.StatusOK, test.same
			})

			want := ""
			if test.want != "" {
				want = covers + test.want + ".jpg"
			}

			got, err := cover(t, address, "Moon Garden")
			if got != want || err != nil {
				t.Errorf("got %q, %v, want %q", got, err, want)
			}

			if got := requests(); !slices.Equal(got, test.requests) {
				t.Errorf("got the requests %q, want %q", got, test.requests)
			}
		})
	}
}

func TestCoverEscapesTheTitle(t *testing.T) {
	address, requests := igdbStandIn(t, games(`[]`))

	got, err := cover(t, address, "Say \"Hi\" \\o/\n\t\x7f")
	if got != "" || err != nil {
		t.Errorf("got %q, %v, want no cover", got, err)
	}

	want := []string{
		asked("token-1", `search "Say \"Hi\" \\o/"; fields name,cover.image_id; `+
			`where game_type = (0,2,4,8,9,10,11) & version_parent = null; limit 1;`),
		asked("token-1", `fields name,cover.image_id; where name ~ "Say \"Hi\" \\o/"; limit 1;`),
	}
	if got := requests(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestCoverOfNoTitleAsksNothing(t *testing.T) {
	address, requests := igdbStandIn(t, games(`[]`))
	noTokens := &tokens{err: errors.New("asked for a token")}

	got, err := igdb.New(address, clientID, noTokens).Cover(t.Context(), "")
	if got != "" || err != nil {
		t.Errorf("got %q, %v, want no cover", got, err)
	}

	if got := requests(); len(got) != 0 {
		t.Errorf("got the requests %q, want none", got)
	}
}
