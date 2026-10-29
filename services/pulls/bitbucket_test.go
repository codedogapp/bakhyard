package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
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
		secretsWith(map[string]string{"bitbucket-token": "secret"}))
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
	client, err := bitbucketClient(envOf(nil), secretsWith(nil))
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
		// No token: the settings are checked before the token is read.
		client, err := bitbucketClient(envOf(test.settings), secretsWith(nil))
		if client != nil || err == nil || err.Error() != test.want {
			t.Errorf("got %v, %v, want the error %s", client, err, test.want)
		}
	}
}

func TestBitbucketClientNeedsTheToken(t *testing.T) {
	client, err := bitbucketClient(envOf(bitbucketSettings(bitbucketAPI)), secretsWith(nil))

	want := "read the Bitbucket token: no secret bitbucket-token"
	if client != nil || err == nil || err.Error() != want {
		t.Errorf("got %v, %v, want the error %s", client, err, want)
	}
}

func TestPullsStopsOnABadBitbucketSetting(t *testing.T) {
	_, err := routes(t.Context(), envOf(bitbucketWith(bitbucketEmailVar, "")), secretsWith(nil))
	if err == nil || err.Error() != "BAKHYARD_BITBUCKET_EMAIL is not set" {
		t.Errorf("got %v, want the error BAKHYARD_BITBUCKET_EMAIL is not set", err)
	}
}

// bitbucketServer returns the address of a stand-in for Bitbucket's API, for the email
// me@example.com with the token secret, and a function that returns the paths it was asked so
// far. It finds me as {me}, and in each of the workspaces acme and labs one open pull request of
// mine, into its repo web, and one that waits for my review, in its only other repo, api or app.
// My pull request into acme/web has no merge check.
func bitbucketServer(t *testing.T) (string, func() []string) {
	t.Helper()

	answers := map[string]string{
		"/2.0/user":                                                     `{"uuid":"{me}"}`,
		"/2.0/workspaces/acme/pullrequests/{me}":                        pageInto("acme/web"),
		"/2.0/workspaces/labs/pullrequests/{me}":                        pageInto("labs/web"),
		"/2.0/repositories/acme":                                        `{"values":[{"full_name":"acme/api"}]}`,
		"/2.0/repositories/labs":                                        `{"values":[{"full_name":"labs/app"}]}`,
		"/2.0/repositories/acme/api/pullrequests":                       pageInto("acme/api"),
		"/2.0/repositories/labs/app/pullrequests":                       pageInto("labs/app"),
		"/2.0/repositories/acme/web/pullrequests/1/mergeability/checks": `{"values":[]}`,
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
