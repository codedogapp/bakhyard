package bitbucket_test

import (
	"encoding/json/v2"
	"testing"
)

// mineOnly returns the pull requests that Mine lists when Bitbucket has request alone.
func mineOnly(t *testing.T, request map[string]any) string {
	t.Helper()

	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		mineIn("acme"): okay(t, pageOf("", request)),
	}}

	got, err := clientOf(fake.serve(t), "acme").Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func TestMineGivesEachPullItsFieldsInUTC(t *testing.T) {
	request := pullRequest(42, "Ship it", "web", 0)
	request["draft"] = true
	request["created_on"] = "2026-02-28T00:30:00.5+03:00"
	request["updated_on"] = "2026-03-01T06:15:00-01:00"

	want := `[{"id":42,"title":"Ship it","repo":"acme/web",` +
		`"url":"https://bitbucket.org/acme/web/pull-requests/42","draft":true,` +
		`"created":"2026-02-27T21:30:00.5Z","updated":"2026-03-01T07:15:00Z"}]`
	if got := mineOnly(t, request); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMineLinksOnlyToHTTPSPages(t *testing.T) {
	for _, href := range []string{
		"javascript:alert(1)",
		"http://bitbucket.org/acme/web/pull-requests/1",
		"//bitbucket.org/acme/web/pull-requests/1",
		"https:relative",
		"https://%zz",
		"",
	} {
		request := pullRequest(1, "One", "web", 1)
		request["links"] = map[string]any{"html": map[string]any{"href": href}}

		want := `[{"id":1,"title":"One","repo":"acme/web","draft":false,` +
			`"created":"2026-03-01T09:00:00.123456Z","updated":"2026-03-01T12:01:00Z"}]`
		if got := mineOnly(t, request); got != want {
			t.Errorf("linked to %q: got %s, want %s", href, got, want)
		}
	}
}

func TestMineReadsAPullWithoutItsLinks(t *testing.T) {
	request := pullRequest(1, "One", "web", 1)
	delete(request, "links")
	delete(request, "destination")

	want := `[{"id":1,"title":"One","repo":"","draft":false,` +
		`"created":"2026-03-01T09:00:00.123456Z","updated":"2026-03-01T12:01:00Z"}]`
	if got := mineOnly(t, request); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
