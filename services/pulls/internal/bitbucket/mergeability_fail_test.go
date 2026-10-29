package bitbucket_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestApprovalsFailOnABadAnswer(t *testing.T) {
	const action = "read the merge checks of acme/web #42: "

	for _, test := range []struct {
		answer reply
		// want starts the error.
		want string
	}{
		{reply{http.StatusBadRequest, `{"type":"error"}`}, action + "400 Bad Request"},
		{reply{http.StatusForbidden, ""}, action + "403 Forbidden"},
		{reply{http.StatusNotFound, `{"type":"error"}`}, action + "404 Not Found"},
		{reply{http.StatusTooManyRequests, ""}, action + "429 Too Many Requests"},
		{reply{http.StatusInternalServerError, ""}, action + "500 Internal Server Error"},
		{reply{http.StatusOK, "<html>Log in</html>"}, "read Bitbucket's answer: "},
		{reply{http.StatusOK, `{"values":{}}`}, "read Bitbucket's answer: "},
	} {
		fake := &fakeBitbucket{pages: map[string]reply{checksOf("acme/web", 42): test.answer}}

		got, err := clientOf(fake.serve(t), "acme").Approvals(t.Context(), myPull())
		if got != (bitbucket.Approvals{}) || err == nil ||
			!strings.HasPrefix(err.Error(), test.want) {
			t.Errorf("%d %q: got %+v, %v, want an error that starts with %q",
				test.answer.status, test.answer.body, got, err, test.want)
		}
	}
}

func TestApprovalsRejectARepoThatIsNotAFullName(t *testing.T) {
	for _, name := range []string{
		"", "acme", "/web", "acme/", ".", "./web", "acme/.", "../web", "acme/..", "../..",
	} {
		fake := &fakeBitbucket{pages: map[string]reply{}}

		got, err := clientOf(fake.serve(t), "acme").Approvals(t.Context(),
			bitbucket.Pull{ID: 42, Repo: name})

		want := fmt.Sprintf("read the merge checks of %s #42: %q is not a repository's full name",
			name, name)
		if requests := fake.requests(); got != (bitbucket.Approvals{}) || err == nil ||
			err.Error() != want || len(requests) != 0 {
			t.Errorf("%q: got %+v, %v after the requests %q, want the error %s before any",
				name, got, err, requests, want)
		}
	}
}

func TestApprovalsEscapeTheWorkspaceAndTheSlug(t *testing.T) {
	fake := &fakeBitbucket{pages: map[string]reply{}}

	_, _ = clientOf(fake.serve(t), "acme").Approvals(t.Context(),
		bitbucket.Pull{ID: 42, Repo: "a b?/c%d#e/f"})

	want := "GET " + checksOf("a%20b%3F/c%25d%23e%2Ff", 42)
	if requests := fake.requests(); len(requests) != 1 || requests[0] != want {
		t.Errorf("got the requests %q, want %q", requests, want)
	}
}
