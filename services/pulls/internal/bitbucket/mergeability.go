package bitbucket

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
)

// checksQuery leaves out the merge checks that are slow to work out and that the approvals don't
// need. Bitbucket's spec allows only these two exclusions here; any other q gets a 400.
const checksQuery = `type!="git_mergeability_check" AND check.kind!="maximum_commits_behind"`

// Approvals are how many approvals a pull request has, and how many it needs.
type Approvals struct {
	Given    int
	Required int
}

// checkType is a merge check's type in Bitbucket's answer, such as standard_merge_check.
type checkType string

// checkKind is a standard merge check's kind in Bitbucket's answer, such as minimum_approvals.
type checkKind string

const (
	// standardCheck is the type of a check that a restriction on the target branch makes.
	standardCheck checkType = "standard_merge_check"
	// minimumApprovals is the kind of the check that asks for a number of approvals.
	minimumApprovals checkKind = "minimum_approvals"
)

// mergeCheck is the part of a merge check in Bitbucket's answer that the client reads.
type mergeCheck struct {
	Type        checkType   `json:"type"`
	Check       definition  `json:"check"`
	Requirement requirement `json:"requirement"`
	Observed    observed    `json:"observed"`
}

// definition says which kind of restriction a standard merge check comes from.
type definition struct {
	Kind checkKind `json:"kind"`
}

// requirement is what a minimum_approvals check asks for.
type requirement struct {
	MinimumApprovals int `json:"minimum_approvals"`
}

// observed is what a minimum_approvals check finds.
type observed struct {
	ApprovalCount int `json:"approval_count"`
}

// Approvals returns how many approvals pull has and how many it needs, from its merge checks
// (decisions 175 and 176). The check of the restriction that requires the most approvals gives
// both. Without one, pull needs one approval, and has those of its reviewers.
func (c *Client) Approvals(ctx context.Context, pull Pull) (Approvals, error) {
	action := fmt.Sprintf("read the merge checks of %s #%d", pull.Repo, pull.ID)

	target, err := c.pullRequestsOf(pull.Repo, action)
	if err != nil {
		return Approvals{}, err
	}

	target = target.JoinPath(strconv.Itoa(pull.ID), "mergeability", "checks")
	target.RawQuery = url.Values{"q": {checksQuery}}.Encode()

	var found page[mergeCheck]

	err = c.get(ctx, target, action, &found)
	if err != nil {
		return Approvals{}, err
	}

	return approvals(found.Values, pull.Reviewers), nil
}

// approvals returns the approvals from the minimum_approvals check among checks that requires the
// most or, without one, one required and as many given as reviewers have approved.
func approvals(checks []mergeCheck, reviewers []Reviewer) Approvals {
	var asked []Approvals

	for _, check := range checks {
		if check.Type == standardCheck && check.Check.Kind == minimumApprovals {
			asked = append(asked, Approvals{
				Given:    check.Observed.ApprovalCount,
				Required: check.Requirement.MinimumApprovals,
			})
		}
	}

	if len(asked) == 0 {
		return Approvals{Given: approved(reviewers), Required: 1}
	}

	return slices.MaxFunc(asked, func(a, b Approvals) int {
		return cmp.Compare(a.Required, b.Required)
	})
}

// approved returns how many of reviewers have approved.
func approved(reviewers []Reviewer) int {
	count := 0

	for _, reviewer := range reviewers {
		if reviewer.Verdict == Approved {
			count++
		}
	}

	return count
}
