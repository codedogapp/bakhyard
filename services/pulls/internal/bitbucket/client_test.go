package bitbucket_test

import (
	"net/http"
	"slices"
	"testing"
)

func TestMineAsksForMyOpenPullsInEachWorkspace(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		mineIn("acme"): okay(t, pageOf("", pullRequest(7, "Export the report", "web", 5))),
		mineIn("beta"): okay(t, pageOf("", pullRequest(3, "Fix the build", "api", 9))),
	}}

	got, err := clientOf(fake.serve(t), "acme", "beta").Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if ids := idsOf(got); !slices.Equal(ids, []int{3, 7}) {
		t.Errorf("got the pull requests %v, want 3 and 7", ids)
	}

	want := []string{"GET /2.0/user", "GET " + mineIn("acme"), "GET " + mineIn("beta")}
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

func TestMineFollowsEachNextLinkAsGiven(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}
	api := fake.serve(t)
	host := api.Scheme + "://" + api.Host

	fake.pages[mineIn("acme")] = okay(t, pageOf(host+"/2.0/opaque?cursor=b%2Fc",
		pullRequest(3, "Three", "web", 9), pullRequest(2, "Two", "web", 8)))
	fake.pages["/2.0/opaque?cursor=b%2Fc"] = okay(t, pageOf(host+"/2.0/last",
		pullRequest(1, "One", "web", 7)))
	fake.pages["/2.0/last"] = okay(t, pageOf("", pullRequest(9, "Nine", "api", 6)))

	got, err := clientOf(api, "acme").Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /2.0/user", "GET " + mineIn("acme"), "GET /2.0/opaque?cursor=b%2Fc", "GET /2.0/last",
	}
	if ids, requests := idsOf(got), fake.requests(); !slices.Equal(ids, []int{3, 2, 1, 9}) ||
		!slices.Equal(requests, want) {
		t.Errorf("got the pull requests %v after the requests %q, want 3, 2, 1 and 9 after %q",
			ids, requests, want)
	}
}

func TestMineListsTheLatestUpdatedFirst(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
		mineIn("acme"): okay(t, pageOf("",
			pullRequest(1, "", "web", 10), pullRequest(2, "", "web", 30),
			pullRequest(3, "", "web", 20), pullRequest(4, "", "web", 30),
			pullRequest(5, "", "web", 10), pullRequest(6, "", "web", 20),
			pullRequest(7, "", "web", 30), pullRequest(8, "", "web", 10))),
		mineIn("beta"): okay(t, pageOf("",
			pullRequest(9, "", "api", 20), pullRequest(10, "", "api", 40),
			pullRequest(11, "", "api", 30), pullRequest(12, "", "api", 20),
			pullRequest(13, "", "api", 40), pullRequest(14, "", "api", 10))),
	}}

	got, err := clientOf(fake.serve(t), "acme", "beta").Mine(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	// Pull requests updated at the same time keep Bitbucket's order, workspace by workspace.
	// There are more than 12, since Go sorts up to 12 elements stably either way.
	want := []int{10, 13, 2, 4, 7, 11, 3, 6, 9, 12, 1, 5, 8, 14}
	if ids := idsOf(got); !slices.Equal(ids, want) {
		t.Errorf("got %v, want %v", ids, want)
	}
}

func TestMineListsNoneWhenIHaveNone(t *testing.T) {
	for _, workspaces := range [][]string{{"acme"}, nil} {
		fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{
			mineIn("acme"): {http.StatusOK, `{"values":[]}`},
		}}

		got, err := clientOf(fake.serve(t), workspaces...).Mine(t.Context())
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("in %q: got %v, %v, want an empty list", workspaces, got, err)
		}
	}
}

func TestMineEscapesTheWorkspace(t *testing.T) {
	fake := &fakeBitbucket{user: meAnswer(t), pages: map[string]reply{}}

	_, _ = clientOf(fake.serve(t), "a/b?c#d%").Mine(t.Context())

	want := "GET " + mineIn("a%2Fb%3Fc%23d%25")
	if requests := fake.requests(); len(requests) != 2 || requests[1] != want {
		t.Errorf("got the requests %q, want the second to be %q", requests, want)
	}
}
