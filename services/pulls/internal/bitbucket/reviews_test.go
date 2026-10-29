package bitbucket_test

import (
	"fmt"
	"net/http"
	"path"
	"slices"
	"testing"
)

// reviewsQuery is the query with which the client lists the open pull requests that wait for my
// review: the filter state="OPEN" AND reviewers.uuid="<myUUID>", form-encoded.
const reviewsQuery = "?pagelen=50&q=state%3D%22OPEN%22+AND+reviewers.uuid%3D%22" +
	"%7B6b0c7a4e-1d2f-4c3b-9a8e-5f6d7c8b9a01%7D%22"

// reposIn returns the URI at which the client asks for workspace's latest updated repositories.
func reposIn(workspace string) string {
	return "/2.0/repositories/" + workspace + "?pagelen=100&sort=-updated_on"
}

// reviewsIn returns the URI at which the client asks for the open pull requests that wait for my
// review in the repository whose full name, its parts escaped, is fullName.
func reviewsIn(fullName string) string {
	return "/2.0/repositories/" + fullName + "/pullrequests" + reviewsQuery
}

// repo returns a repository as a workspace's list holds it, with fields the client doesn't read
// too. It was last updated minute minutes past noon UTC on 1 March 2026.
func repo(fullName string, minute int) map[string]any {
	return map[string]any{
		"type": "repository", "full_name": fullName, "name": path.Base(fullName),
		"updated_on": fmt.Sprintf("2026-03-01T12:%02d:00.000000+00:00", minute),
		"is_private": true,
	}
}

// none is a page with no values.
var none = reply{http.StatusOK, `{"values":[]}`}

func TestReviewsAsksEachLatestRepositoryForThePullsThatWaitForMe(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		reposIn("acme"):       okay(t, pageOf("", repo("acme/web", 30), repo("acme/api", 10))),
		reposIn("beta"):       okay(t, pageOf("", repo("beta/app", 20))),
		reviewsIn("acme/web"): okay(t, pageOf("", pullRequest(7, "Export the report", "web", 5))),
		reviewsIn("beta/app"): okay(t, pageOf("", pullRequest(3, "Fix the build", "app", 9))),
		reviewsIn("acme/api"): none,
	}}

	got, err := clientOf(fake.serve(t), "acme", "beta").Reviews(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if ids := idsOf(got); !slices.Equal(ids, []int{3, 7}) {
		t.Errorf("got the pull requests %v, want 3 and 7, the latest updated first", ids)
	}

	want := []string{
		"GET /2.0/user", "GET " + reposIn("acme"), "GET " + reposIn("beta"),
		"GET " + reviewsIn("acme/web"), "GET " + reviewsIn("beta/app"),
		"GET " + reviewsIn("acme/api"),
	}
	if requests := fake.requests(); !slices.Equal(requests, want) {
		t.Errorf("got the requests %q, want %q", requests, want)
	}

	for _, request := range fake.notes() {
		if request.user != email || request.password != token ||
			request.accept != "application/json" {
			t.Errorf("%s: got the user %q, the password %q and Accept %q, want the token's",
				request.request, request.user, request.password, request.accept)
		}
	}
}

func TestReviewsLooksOnlyInTheHundredLatestUpdatedRepositories(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)
	later := api.Scheme + "://" + api.Host + "/2.0/later"
	fake.pages["/2.0/later"] = okay(t, pageOf("", repo("acme/old", 0)))

	pages := map[string][]map[string]any{}

	for minute := 59; minute >= 0; minute-- {
		for _, workspace := range []string{"acme", "beta"} {
			name := fmt.Sprintf("%s/r%d", workspace, minute)
			pages[workspace] = append(pages[workspace], repo(name, minute))
			fake.pages[reviewsIn(name)] = none
		}
	}

	fake.pages[reposIn("acme")] = okay(t, pageOf(later, pages["acme"]...))
	fake.pages[reposIn("beta")] = okay(t, pageOf("", pages["beta"]...))

	got, err := clientOf(api, "acme", "beta").Reviews(t.Context())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %v, %v, want an empty list", got, err)
	}

	// The repositories updated at the same time keep their order, workspace by workspace, and the
	// next page of acme's is never asked for.
	want := []string{"GET /2.0/user", "GET " + reposIn("acme"), "GET " + reposIn("beta")}
	for minute := 59; minute >= 10; minute-- {
		want = append(want, "GET "+reviewsIn(fmt.Sprintf("acme/r%d", minute)),
			"GET "+reviewsIn(fmt.Sprintf("beta/r%d", minute)))
	}

	if requests := fake.requests(); !slices.Equal(requests, want) {
		t.Errorf("got the requests %q, want %q", requests, want)
	}
}

func TestReviewsFollowsEachNextLinkOfThePulls(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)
	host := api.Scheme + "://" + api.Host

	fake.pages[reposIn("acme")] = okay(t, pageOf("", repo("acme/web", 1)))
	fake.pages[reviewsIn("acme/web")] = okay(t, pageOf(host+"/2.0/opaque?cursor=b%2Fc",
		pullRequest(3, "Three", "web", 9), pullRequest(2, "Two", "web", 8)))
	fake.pages["/2.0/opaque?cursor=b%2Fc"] = okay(t, pageOf("", pullRequest(1, "One", "web", 7)))

	got, err := clientOf(api, "acme").Reviews(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /2.0/user", "GET " + reposIn("acme"), "GET " + reviewsIn("acme/web"),
		"GET /2.0/opaque?cursor=b%2Fc",
	}
	if ids, requests := idsOf(got), fake.requests(); !slices.Equal(ids, []int{3, 2, 1}) ||
		!slices.Equal(requests, want) {
		t.Errorf("got the pull requests %v after the requests %q, want 3, 2 and 1 after %q",
			ids, requests, want)
	}
}

func TestReviewsListsNoneWhenNoneWaits(t *testing.T) {
	for _, test := range []struct {
		name       string
		workspaces []string
		repos      reply
	}{
		{"no workspace", nil, none},
		{"no repository", []string{"acme"}, none},
		{"no pull request", []string{"acme"}, okay(t, pageOf("", repo("acme/web", 1)))},
	} {
		fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
			reposIn("acme"): test.repos, reviewsIn("acme/web"): none,
		}}

		got, err := clientOf(fake.serve(t), test.workspaces...).Reviews(t.Context())
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("%s: got %v, %v, want an empty list", test.name, got, err)
		}
	}
}

func TestReviewsEscapesTheWorkspaceAndTheRepository(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	fake.pages[reposIn("a%2Fb%3Fc%23d%25")] = okay(t, pageOf("", repo("a b?/c#d%/e", 1)))

	_, _ = clientOf(fake.serve(t), "a/b?c#d%").Reviews(t.Context())

	want := []string{
		"GET /2.0/user", "GET " + reposIn("a%2Fb%3Fc%23d%25"),
		"GET " + reviewsIn("a%20b%3F/c%23d%25%2Fe"),
	}
	if requests := fake.requests(); !slices.Equal(requests, want) {
		t.Errorf("got the requests %q, want %q", requests, want)
	}
}
