package github_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// onlyItem returns the item that Mine lists when GitHub finds found alone.
func onlyItem(t *testing.T, found map[string]any) github.Item {
	t.Helper()

	fake := &fakeGitHub{pages: map[string]reply{searchFor(mine, 1): okay(t, resultsOf(found))}}

	got, err := clientOf(fake.serve(t)).Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d items, want one", len(got))
	}

	return got[0]
}

// onlyJSON returns, as JSON, the item that Mine lists when GitHub finds found alone.
func onlyJSON(t *testing.T, found map[string]any) string {
	t.Helper()

	encoded, err := json.Marshal(onlyItem(t, found))
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func TestItemHasItsFieldsInUTC(t *testing.T) {
	found := pullIn("web", 42)
	found["title"] = "Ship it"
	found["draft"] = true
	found["created_at"] = "2026-03-01T09:00:00+01:00"
	found["updated_at"] = "2026-03-01T12:05:00.5-02:00"

	want := `{"id":42,"title":"Ship it","repo":"acme/web",` +
		`"url":"https://github.com/acme/web/pull/42","draft":true,` +
		`"created":"2026-03-01T08:00:00Z","updated":"2026-03-01T14:05:00.5Z"}`
	if got := onlyJSON(t, found); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestAnIssueIsNoDraft(t *testing.T) {
	found := pullIn("web", 8)
	delete(found, "draft")
	delete(found, "pull_request")
	found["html_url"] = "https://github.com/acme/web/issues/8"

	want := `{"id":8,"title":"Pull 8","repo":"acme/web",` +
		`"url":"https://github.com/acme/web/issues/8","draft":false,` +
		`"created":"2026-03-01T09:00:00Z","updated":"2026-03-01T12:00:00Z"}`
	if got := onlyJSON(t, found); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestItemTakesItsRepoFromItsAddressInTheAPI(t *testing.T) {
	for address, want := range map[string]string{
		"https://api.github.com/repos/codedogapp/bakhyard": "codedogapp/bakhyard",
		"https://api.github.com/repos/repos/repos":         "repos/repos",
		"https://ghe.example/api/v3/repos/acme/web":        "acme/web",
	} {
		found := pullIn("web", 1)
		found["repository_url"] = address

		if got := onlyItem(t, found).Repo; got != want {
			t.Errorf("%s: got %q, want %q", address, got, want)
		}
	}
}

func TestItemLinksOnlyAnHTTPSPage(t *testing.T) {
	for _, page := range []string{
		"http://github.com/acme/web/pull/1", "javascript:alert(1)", "https:///acme/web/pull/1",
		"/acme/web/pull/1", "", "%zz",
	} {
		found := pullIn("web", 1)
		found["html_url"] = page

		want := `{"id":1,"title":"Pull 1","repo":"acme/web","draft":false,` +
			`"created":"2026-03-01T09:00:00Z","updated":"2026-03-01T12:00:00Z"}`
		if got := onlyJSON(t, found); got != want {
			t.Errorf("%q: got %s, want %s", page, got, want)
		}
	}
}
