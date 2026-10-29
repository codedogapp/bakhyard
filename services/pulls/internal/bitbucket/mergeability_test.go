package bitbucket_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// checksOf returns the URI at which the client asks for the merge checks of pull request id in
// the repository whose path is repo, such as acme/web, leaving out the slow ones.
func checksOf(repo string, id int) string {
	return "/2.0/repositories/" + repo + "/pullrequests/" + strconv.Itoa(id) +
		"/mergeability/checks?q=type%21%3D%22git_mergeability_check%22+AND+" +
		"check.kind%21%3D%22maximum_commits_behind%22"
}

// check returns a standard merge check of kind as Bitbucket's answers hold it, failed and
// blocking, with fields the client doesn't read too.
func check(kind string, requirement, observed map[string]any) map[string]any {
	return map[string]any{
		"type": "standard_merge_check", "status": "FAILED", "required": true, "blocking": true,
		"check":       map[string]any{"type": "standard_merge_check_definition", "kind": kind},
		"requirement": requirement, "observed": observed,
	}
}

// approvalsCheck returns a minimum_approvals check that asks for required approvals and finds
// given, passed when given is enough.
func approvalsCheck(required, given int) map[string]any {
	found := check("minimum_approvals",
		map[string]any{"minimum_approvals": required}, map[string]any{"approval_count": given})
	if given >= required {
		found["status"], found["blocking"] = "PASSED", false
	}

	return found
}

// unenforced returns found as Bitbucket answers it for a repository that doesn't enforce its
// merge checks: not required, and so not blocking.
func unenforced(found map[string]any) map[string]any {
	found["required"], found["blocking"] = false, false

	return found
}

// otherChecks are merge checks that don't count: shaped as a public pull request's were on
// 2026-10-29, the queue's as Bitbucket's spec has it, and a made-up custom check that carries a
// minimum_approvals check's kind and fields.
var otherChecks = []map[string]any{
	{
		"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false,
		"state": "OPEN",
	},
	{
		"type": "current_user_permission_check", "status": "PASSED", "required": true,
		"blocking": false,
	},
	check("minimum_default_reviewer_approvals",
		map[string]any{"minimum_default_reviewer_approvals": 3},
		map[string]any{"default_reviewer_approval_count": 0}),
	check("no_changes_requested",
		map[string]any{"changes_requested_allowed": false},
		map[string]any{"changes_requested": true}),
	check("minimum_successful_builds",
		map[string]any{"minimum_successful_builds": 1},
		map[string]any{"successful_build_count": 0}),
	check("resolved_tasks",
		map[string]any{"unresolved_tasks_allowed": false},
		map[string]any{"unresolved_task_count": 2}),
	{
		"type": "custom_pre_merge_check", "status": "FAILED", "required": true, "blocking": true,
		"check": map[string]any{
			"type": "custom_merge_check_definition", "id": "ari:x", "name": "Approvals",
			"kind": "minimum_approvals",
		},
		"requirement": map[string]any{"minimum_approvals": 4},
		"observed":    map[string]any{"approval_count": 0},
	},
	{
		"type": "merge_queue_check", "status": "PASSED", "required": false, "blocking": false,
		"queued":      false,
		"merge_queue": map[string]any{"uuid": "{q}", "name": "main", "state": "ACTIVE"},
	},
}

// approvalsOf returns the approvals of a pull request that has given and needs required.
func approvalsOf(given, required int) bitbucket.Approvals {
	return bitbucket.Approvals{Given: given, Required: required}
}

// myPull returns my pull request 42 into acme/web, which reviewers review.
func myPull(reviewers ...bitbucket.Reviewer) bitbucket.Pull {
	return bitbucket.Pull{ID: 42, Title: "Ship it", Repo: "acme/web", Reviewers: reviewers}
}

// approvalsWith returns what Approvals gives for pull when Bitbucket answers its merge checks
// with checks.
func approvalsWith(
	t *testing.T, pull bitbucket.Pull, checks ...map[string]any,
) (bitbucket.Approvals, error) {
	t.Helper()

	fake := &fakeBitbucket{pages: map[string]reply{
		checksOf("acme/web", 42): okay(t, map[string]any{"size": len(checks), "values": checks}),
	}}

	return clientOf(fake.serve(t), "acme").Approvals(t.Context(), pull)
}

func TestApprovalsAsksForThePullsMergeChecks(t *testing.T) {
	fake := &fakeBitbucket{pages: map[string]reply{
		checksOf("beta/api", 7): okay(t, map[string]any{
			"size": 1, "values": []map[string]any{approvalsCheck(2, 1)},
		}),
	}}

	got, err := clientOf(fake.serve(t), "acme").Approvals(t.Context(),
		bitbucket.Pull{ID: 7, Repo: "beta/api"})

	want := bitbucket.Approvals{Given: 1, Required: 2}
	if err != nil || got != want {
		t.Errorf("got %+v, %v, want %+v", got, err, want)
	}

	notes := fake.notes()
	if len(notes) != 1 || notes[0] != (asked{
		"GET " + checksOf("beta/api", 7), email, token, "application/json",
	}) {
		t.Errorf("got the requests %+v, want one for the merge checks, with the token", notes)
	}
}

func TestApprovalsComeFromTheMinimumApprovalsCheck(t *testing.T) {
	ana := bitbucket.Reviewer{Name: "Ana", Verdict: bitbucket.Approved}

	for _, test := range []struct {
		name   string
		checks []map[string]any
		want   bitbucket.Approvals
	}{
		{"too few", []map[string]any{approvalsCheck(2, 1)}, approvalsOf(1, 2)},
		{"enough", []map[string]any{approvalsCheck(2, 3)}, approvalsOf(3, 2)},
		{"none required", []map[string]any{approvalsCheck(0, 0)}, approvalsOf(0, 0)},
		{"not enforced", []map[string]any{unenforced(approvalsCheck(2, 1))}, approvalsOf(1, 2)},
		{"the most required, with its count", []map[string]any{
			approvalsCheck(1, 3), approvalsCheck(3, 2), approvalsCheck(2, 0),
		}, approvalsOf(2, 3)},
		{"among the other checks", slices.Concat(
			otherChecks[:3], []map[string]any{approvalsCheck(2, 0)}, otherChecks[3:],
		), approvalsOf(0, 2)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := approvalsWith(t, myPull(ana), test.checks...)
			if err != nil || got != test.want {
				t.Errorf("got %+v, %v, want %+v", got, err, test.want)
			}
		})
	}
}

func TestApprovalsNeedOneWithoutAMinimumApprovalsCheck(t *testing.T) {
	reviewer := func(name string, verdict bitbucket.Verdict) bitbucket.Reviewer {
		return bitbucket.Reviewer{Name: name, Verdict: verdict}
	}

	for _, test := range []struct {
		name   string
		pull   bitbucket.Pull
		checks []map[string]any
		want   bitbucket.Approvals
	}{
		{"no checks, no reviewers", myPull(), nil, approvalsOf(0, 1)},
		{"the other checks", myPull(), otherChecks, approvalsOf(0, 1)},
		{"reviewers who approved", myPull(
			reviewer("Ana", bitbucket.Approved), reviewer("Ben", bitbucket.Pending),
			reviewer("Cy", bitbucket.ChangesRequested), reviewer("Di", bitbucket.Approved),
		), otherChecks, approvalsOf(2, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := approvalsWith(t, test.pull, test.checks...)
			if err != nil || got != test.want {
				t.Errorf("got %+v, %v, want %+v", got, err, test.want)
			}
		})
	}
}
