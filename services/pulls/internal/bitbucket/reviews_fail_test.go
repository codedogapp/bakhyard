package bitbucket_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// The actions that the errors of Reviews's own listings name.
const (
	listingRepos   = "list the Bitbucket repositories"
	listingReviews = "list the Bitbucket pull requests that wait for my review"
)

// reviewsWith returns what Reviews gives when Bitbucket answers uri with answer, and the rest
// well: acme's only repository is web, in which no pull request waits for me.
func reviewsWith(t *testing.T, uri string, answer reply) ([]bitbucket.Pull, error) {
	t.Helper()

	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		reposIn("acme"): okay(t, pageOf("", repo("acme/web", 1))), reviewsIn("acme/web"): none,
	}}
	if uri == "/2.0/user" {
		fake.user = answer
	} else {
		fake.pages[uri] = answer
	}

	return clientOf(fake.serve(t), "acme").Reviews(t.Context())
}

func TestReviewsFailsOnABadAnswer(t *testing.T) {
	for _, step := range []struct{ uri, action string }{
		{"/2.0/user", "find me on Bitbucket"},
		{reposIn("acme"), listingRepos},
		{reviewsIn("acme/web"), listingReviews},
	} {
		for _, test := range []struct {
			answer reply
			// want starts the error, after the failed action when it starts with a colon.
			want string
		}{
			{reply{http.StatusUnauthorized, ""}, ": 401 Unauthorized"},
			{reply{http.StatusNotFound, `{"type":"error"}`}, ": 404 Not Found"},
			{reply{http.StatusOK, "<html>Log in</html>"}, "read Bitbucket's answer: "},
		} {
			got, err := reviewsWith(t, step.uri, test.answer)

			want := test.want
			if strings.HasPrefix(want, ":") {
				want = step.action + want
			}

			if !failed(got, err, want) {
				t.Errorf("%s answered %q: got %v, %v, want an error that starts with %q",
					step.uri, test.answer.body, got, err, want)
			}
		}
	}
}

func TestReviewsFailsWholeWhenAListFails(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		reposIn("acme"): okay(t, pageOf("",
			repo("acme/web", 30), repo("acme/api", 20), repo("acme/doc", 10))),
		reviewsIn("acme/web"): okay(t, pageOf("", pullRequest(7, "Export the report", "web", 5))),
		reviewsIn("acme/api"): {http.StatusInternalServerError, ""},
		reviewsIn("acme/doc"): none,
	}}

	got, err := clientOf(fake.serve(t), "acme").Reviews(t.Context())

	want := []string{
		"GET /2.0/user", "GET " + reposIn("acme"), "GET " + reviewsIn("acme/web"),
		"GET " + reviewsIn("acme/api"),
	}

	requests := fake.requests()
	if !failed(got, err, listingReviews+": 500 Internal Server Error") ||
		!slices.Equal(requests, want) {
		t.Errorf("got %v, %v after the requests %q, want a 500 after %q", got, err, requests,
			want)
	}
}

func TestReviewsRejectsAFullNameThatIsNotARepository(t *testing.T) {
	for _, name := range []string{
		"", "acme", "/web", "acme/", ".", "./web", "acme/.", "../web", "acme/..", "../..",
	} {
		fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
			reposIn("acme"): okay(t, pageOf("", repo(name, 1))),
		}}

		got, err := clientOf(fake.serve(t), "acme").Reviews(t.Context())

		want := fmt.Sprintf("%s: %q is not a repository's full name", listingRepos, name)
		if requests := fake.requests(); got != nil || err == nil || err.Error() != want ||
			len(requests) != 2 {
			t.Errorf("%q: got %v, %v after the requests %q, want the error %s after two",
				name, got, err, requests, want)
		}
	}
}

func TestReviewsNamesItsListingWhenThePagesGoWrong(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)
	again := api.Scheme + "://" + api.Host + "/2.0/again"
	fake.pages[reposIn("acme")] = okay(t, pageOf("", repo("acme/web", 1)))

	for link, want := range map[string]string{
		again:                       listingReviews + ": more than 10 pages",
		"https://" + api.Host + "/": listingReviews + ": the next page is off " + api.Host,
	} {
		fake.pages[reviewsIn("acme/web")] = okay(t, pageOf(link, pullRequest(1, "One", "web", 1)))
		fake.pages["/2.0/again"] = fake.pages[reviewsIn("acme/web")]

		got, err := clientOf(api, "acme").Reviews(t.Context())
		if got != nil || err == nil || err.Error() != want {
			t.Errorf("next %s: got %v, %v, want the error %s", link, got, err, want)
		}
	}
}
