package bitbucket_test

import (
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// person returns an account as Bitbucket's answers hold it, with fields the client doesn't read
// too.
func person(uuid, name string) map[string]any {
	return map[string]any{
		"type": "user", "uuid": uuid, "display_name": name, "nickname": name,
		"account_id": "557058:" + name,
	}
}

// participant returns a pull request's participant as Bitbucket's answers hold it, with fields
// the client doesn't read too.
func participant(user map[string]any, role string, approved bool, state any) map[string]any {
	return map[string]any{
		"type": "participant", "user": user, "role": role, "approved": approved,
		"state": state, "participated_on": nil,
	}
}

// reviewedPull returns a pull request of mine as Bitbucket's pages hold it, with its participants.
func reviewedPull(participants ...map[string]any) map[string]any {
	request := pullRequest(42, "Ship it", "web", 0)
	request["author"] = person(myUUID, "You")
	request["participants"] = participants

	return request
}

func TestMineGivesEachPullItsReviewers(t *testing.T) {
	ana, ben, author := person("{a1}", "Ana"), person("{b2}", "Ben"), person(myUUID, "You")
	rovo := person("{c3}", "Rovo Dev")
	rovo["type"], rovo["kind"], rovo["ai_agent"] = "app_user", "atlassian_app", true

	type reviewers = []bitbucket.Reviewer

	for _, test := range []struct {
		name         string
		participants []map[string]any
		want         reviewers
	}{
		{"a reviewer who hasn't acted", []map[string]any{
			participant(ana, "REVIEWER", false, nil),
		}, reviewers{{Name: "Ana", Verdict: bitbucket.Pending}}},
		{"a reviewer who approved", []map[string]any{
			participant(ana, "REVIEWER", true, "approved"),
		}, reviewers{{Name: "Ana", Verdict: bitbucket.Approved}}},
		{"a reviewer who requested changes", []map[string]any{
			participant(ana, "REVIEWER", false, "changes_requested"),
		}, reviewers{{Name: "Ana", Verdict: bitbucket.ChangesRequested}}},
		{"an approval, whatever the state says", []map[string]any{
			participant(ana, "REVIEWER", true, "changes_requested"),
		}, reviewers{{Name: "Ana", Verdict: bitbucket.Approved}}},
		{"a state the client doesn't know", []map[string]any{
			participant(ana, "REVIEWER", false, "declined"),
		}, reviewers{{Name: "Ana", Verdict: bitbucket.Pending}}},
		{"someone who approved unasked", []map[string]any{
			participant(ben, "PARTICIPANT", true, "approved"),
		}, reviewers{{Name: "Ben", Verdict: bitbucket.Approved}}},
		{"someone who requested changes unasked", []map[string]any{
			participant(ben, "PARTICIPANT", false, "changes_requested"),
		}, reviewers{{Name: "Ben", Verdict: bitbucket.ChangesRequested}}},
		{"someone who only commented", []map[string]any{
			participant(ben, "PARTICIPANT", false, nil),
			participant(rovo, "PARTICIPANT", false, nil),
			participant(person("{d4}", "Cy"), "", false, nil),
		}, nil},
		{"the author, me", []map[string]any{
			participant(author, "PARTICIPANT", true, "approved"),
			participant(author, "REVIEWER", false, "changes_requested"),
		}, nil},
		{"everyone, in Bitbucket's order", []map[string]any{
			participant(ben, "REVIEWER", false, "changes_requested"),
			participant(author, "PARTICIPANT", true, "approved"),
			participant(rovo, "PARTICIPANT", false, nil),
			participant(ana, "REVIEWER", false, nil),
			participant(rovo, "PARTICIPANT", true, "approved"),
		}, reviewers{
			{Name: "Ben", Verdict: bitbucket.ChangesRequested},
			{Name: "Ana", Verdict: bitbucket.Pending},
			{Name: "Rovo Dev", Verdict: bitbucket.Approved},
		}},
		{"no one", nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pullsOnly(t, reviewedPull(test.participants...))
			if len(got) != 1 || !slices.Equal(got[0].Reviewers, test.want) {
				t.Errorf("got %+v, want one pull with the reviewers %+v", got, test.want)
			}
		})
	}
}

func TestMineGivesTheReviewersAsJSON(t *testing.T) {
	request := reviewedPull(
		participant(person("{a1}", "Ana"), "REVIEWER", true, "approved"),
		participant(person("{b2}", "Ben"), "REVIEWER", false, "changes_requested"),
		participant(person("{c3}", "Cy"), "REVIEWER", false, nil),
	)

	want := `[{"id":42,"title":"Ship it","repo":"acme/web",` +
		`"url":"https://bitbucket.org/acme/web/pull-requests/42","draft":false,` +
		`"created":"2026-03-01T09:00:00.123456Z","updated":"2026-03-01T12:00:00Z",` +
		`"reviewers":[{"name":"Ana","verdict":"approved"},` +
		`{"name":"Ben","verdict":"changes_requested"},{"name":"Cy","verdict":"pending"}]}]`
	if got := mineOnly(t, request); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestMineLeavesTheReviewersOutWhenNoOneReviews(t *testing.T) {
	request := reviewedPull(participant(person("{b2}", "Ben"), "PARTICIPANT", false, nil))

	want := `[{"id":42,"title":"Ship it","repo":"acme/web",` +
		`"url":"https://bitbucket.org/acme/web/pull-requests/42","draft":false,` +
		`"created":"2026-03-01T09:00:00.123456Z","updated":"2026-03-01T12:00:00Z"}]`
	if got := mineOnly(t, request); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
