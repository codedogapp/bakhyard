package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

// bitbucketAPI is the Bitbucket API in the settings of most tests.
const bitbucketAPI = "https://api.bitbucket.org/2.0"

func TestBitbucketClientListsEachWorkspaceInTheSettingsOnce(t *testing.T) {
	address, asked := bitbucketServer(t)
	settings := bitbucketSettings(address)
	settings[bitbucketWorkspacesVar] = " labs ,acme,labs"

	client, err := bitbucketClient(envOf(settings),
		keychainWith(map[string]string{"bitbucket-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	pulls, err := client.Mine(t.Context())
	if err != nil || len(pulls) != 2 {
		t.Fatalf("got %v, %v, want a pull request in each workspace", pulls, err)
	}

	want := []string{
		"/2.0/user", "/2.0/workspaces/acme/pullrequests/{me}",
		"/2.0/workspaces/labs/pullrequests/{me}",
	}
	if got := asked(); !slices.Equal(got, want) {
		t.Errorf("got the requests %q, want %q", got, want)
	}
}

func TestBitbucketClientIsOffWithoutTheSettings(t *testing.T) {
	client, err := bitbucketClient(envOf(nil), keychainWith(nil))
	if client != nil || err != nil {
		t.Errorf("got %v, %v, want no client and no error", client, err)
	}
}

func TestBitbucketClientRejects(t *testing.T) {
	for _, test := range []rejection{
		{map[string]string{bitbucketAPIVar: bitbucketAPI}, "BAKHYARD_BITBUCKET_EMAIL is not set"},
		{map[string]string{bitbucketEmailVar: "me@example.com"}, "BAKHYARD_BITBUCKET_API is not set"},
		{map[string]string{bitbucketWorkspacesVar: "acme"}, "BAKHYARD_BITBUCKET_API is not set"},
		{bitbucketWith(bitbucketAPIVar, ""), "BAKHYARD_BITBUCKET_API is not set"},
		{bitbucketWith(bitbucketEmailVar, ""), "BAKHYARD_BITBUCKET_EMAIL is not set"},
		{bitbucketWith(bitbucketWorkspacesVar, ""), "BAKHYARD_BITBUCKET_WORKSPACES is not set"},
		// internal/env tests which URLs pass; this shows that the API's must.
		{
			bitbucketWith(bitbucketAPIVar, "http://api.bitbucket.org/2.0"),
			`BAKHYARD_BITBUCKET_API: "http://api.bitbucket.org/2.0" is not an https URL`,
		},
		notSlugs("acme,,labs"), notSlugs("acme,"), notSlugs(" , "), notSlugs("."),
		notSlugs("acme, .."),
	} {
		// No token: the settings are checked before the Keychain is read.
		client, err := bitbucketClient(envOf(test.settings), keychainWith(nil))
		if client != nil || err == nil || err.Error() != test.want {
			t.Errorf("got %v, %v, want the error %s", client, err, test.want)
		}
	}
}

func TestBitbucketClientNeedsTheToken(t *testing.T) {
	client, err := bitbucketClient(envOf(bitbucketSettings(bitbucketAPI)), keychainWith(nil))

	want := "read the Bitbucket token: no Keychain item bakhyard/bitbucket-token"
	if client != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %s", client, err, want)
	}
}

func TestPullsListsMyPullsInEachWorkspace(t *testing.T) {
	address, _ := bitbucketServer(t)

	rec := getMine(t.Context(), t, address)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"repo":"acme/web"`) ||
		!strings.Contains(rec.Body.String(), `"repo":"labs/web"`) {
		t.Errorf("got %d %q, want a pull request in acme and in labs", rec.Code, rec.Body)
	}
}

func TestPullsStopsListingWhenItStops(t *testing.T) {
	stopped, stop := context.WithCancel(t.Context())
	stop()

	// Nothing listens on port 1, so a listing that went ahead would fail.
	rec := getMine(stopped, t, "http://127.0.0.1:1/2.0")
	if rec.Code != http.StatusOK || rec.Body.String() != "{}" {
		t.Errorf("got %d %s, want 200 {}: no listing, and no failure", rec.Code, rec.Body)
	}
}

func TestPullsStopsOnABadBitbucketSetting(t *testing.T) {
	_, err := routes(t.Context(), envOf(bitbucketWith(bitbucketEmailVar, "")), keychainWith(nil))
	if err == nil || err.Error() != "BAKHYARD_BITBUCKET_EMAIL is not set" {
		t.Errorf("got %v, want the error BAKHYARD_BITBUCKET_EMAIL is not set", err)
	}
}

// bitbucketServer returns the address of a stand-in for Bitbucket's API, for the email
// me@example.com with the token secret, and a function that returns the paths it was asked so
// far. It finds me as {me}, and one open pull request of mine in each of the workspaces acme and
// labs, into its repo web.
func bitbucketServer(t *testing.T) (string, func() []string) {
	t.Helper()

	answers := map[string]string{
		"/2.0/user":                              `{"uuid":"{me}"}`,
		"/2.0/workspaces/acme/pullrequests/{me}": pageInto("acme/web"),
		"/2.0/workspaces/labs/pullrequests/{me}": pageInto("labs/web"),
	}

	var (
		lock  sync.Mutex
		paths []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		paths = append(paths, r.URL.Path)

		email, token, ok := r.BasicAuth()
		if !ok || email != "me@example.com" || token != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		answer, found := answers[r.URL.Path]
		if !found {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	return server.URL + "/2.0", func() []string {
		lock.Lock()
		defer lock.Unlock()

		return slices.Clone(paths)
	}
}

// pageInto returns a page of one open pull request, into the repo whose full name is repo.
func pageInto(repo string) string {
	return `{"values":[{"id":1,"title":"Ship","destination":{"repository":{"full_name":"` +
		repo + `"}}}]}`
}

// getMine returns the answer to GET /pulls/mine of pulls with Bitbucket's API at address, whose
// routes run until ctx ends.
func getMine(ctx context.Context, t *testing.T, address string) *httptest.ResponseRecorder {
	t.Helper()

	router, err := routes(ctx, envOf(bitbucketSettings(address)),
		keychainWith(map[string]string{"bitbucket-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pulls/mine", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

// bitbucketSettings returns the Bitbucket settings, with the API at address, for the workspaces
// acme and labs.
func bitbucketSettings(address string) map[string]string {
	return map[string]string{
		bitbucketAPIVar: address, bitbucketEmailVar: "me@example.com",
		bitbucketWorkspacesVar: "acme,labs",
	}
}

// bitbucketWith returns the Bitbucket settings, with name set to value.
func bitbucketWith(name, value string) map[string]string {
	settings := bitbucketSettings(bitbucketAPI)
	settings[name] = value

	return settings
}

// rejection is a row of TestBitbucketClientRejects: the settings, and the error they give.
type rejection struct {
	settings map[string]string
	want     string
}

// notSlugs returns the rejection of value in the workspaces' setting as not a list of slugs.
func notSlugs(value string) rejection {
	return rejection{
		bitbucketWith(bitbucketWorkspacesVar, value),
		bitbucketWorkspacesVar + `: "` + value + `" is not a list of workspace slugs`,
	}
}
