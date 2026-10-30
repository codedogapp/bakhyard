package exophase_test

import (
	"encoding/json/v2"
	"strconv"
	"testing"
	"time"
)

func TestPlayedSinceAnswersEachGame(t *testing.T) {
	played := strconv.FormatInt(now.Add(-time.Hour).Unix(), 10)

	for _, test := range []struct {
		name string
		// listed are the fields of the game in Exophase's list, besides when it was last played.
		listed string
		// want is the game in the answer of the games service.
		want string
	}{
		{
			"trophies",
			`"earned_awards":12,"total_awards":48,"percent":25.5,"playtime":"7h 15m",` +
				`"playtimeUnits":{"hours":7,"minutes":15},` +
				`"meta":{"title":"Moon Garden","platforms":[{"name":"PS4"},{"name":"PS5"}]}`,
			`{"title":"Moon Garden","platforms":["PS4","PS5"],"lastPlayed":"2026-10-30T11:00:00Z",` +
				`"playtimeMinutes":435,"trophies":{"earned":12,"total":48,"percent":25.5}}`,
		},
		{
			"no trophy earned yet",
			`"earned_awards":0,"total_awards":30,"percent":0,` +
				`"playtimeUnits":{"hours":0,"minutes":20},` +
				`"meta":{"title":"Tower Climb","platforms":[{"name":"PS5"}]}`,
			`{"title":"Tower Climb","platforms":["PS5"],"lastPlayed":"2026-10-30T11:00:00Z",` +
				`"playtimeMinutes":20,"trophies":{"earned":0,"total":30,"percent":0}}`,
		},
		{
			"no trophies, as on Nintendo's",
			`"earned_awards":0,"total_awards":0,"percent":0,` +
				`"playtimeUnits":{"hours":1,"minutes":5},` +
				`"meta":{"title":"Kart Party","platforms":[{"name":"Switch"}]}`,
			`{"title":"Kart Party","platforms":["Switch"],"lastPlayed":"2026-10-30T11:00:00Z",` +
				`"playtimeMinutes":65}`,
		},
		{
			"no trophies but a percentage",
			`"earned_awards":0,"total_awards":0,"percent":100,"meta":{"title":"A"}`,
			`{"title":"A","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z"}`,
		},
		{
			"a playtime in fractions",
			`"playtimeUnits":{"hours":1.5,"minutes":0.4},"meta":{"title":"A"}`,
			`{"title":"A","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z","playtimeMinutes":90}`,
		},
		{
			"half a minute",
			`"playtimeUnits":{"hours":0,"minutes":29.5},"meta":{"title":"A"}`,
			`{"title":"A","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z","playtimeMinutes":30}`,
		},
		{
			"an unknown playtime",
			`"playtime":"","playtimeUnits":{"hours":0,"minutes":0},"meta":{"title":"A"}`,
			`{"title":"A","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z"}`,
		},
		{
			"a playtime below 0",
			`"playtimeUnits":{"hours":-1,"minutes":0},"meta":{"title":"A"}`,
			`{"title":"A","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z"}`,
		},
		{
			"nulls",
			`"earned_awards":null,"total_awards":null,"percent":null,"playtimeUnits":null,` +
				`"meta":{"title":null,"platforms":null}`,
			`{"title":"","platforms":[],"lastPlayed":"2026-10-30T11:00:00Z"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeExophase{pages: map[string]reply{
				pageURI(1): ok(pageOf(`{"lastplayed_utc":` + played + "," + test.listed + "}")),
			}}

			got, err := list(t, fake)
			if err != nil {
				t.Fatal(err)
			}

			// The answer is the same whatever the time zone of the machine it runs on.
			if len(got) != 1 || got[0].LastPlayed.Location() != time.UTC {
				t.Fatalf("got %v, want one game last played in UTC", got)
			}

			answer, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}

			if want := "[" + test.want + "]"; string(answer) != want {
				t.Errorf("got %s, want %s", answer, want)
			}
		})
	}
}
