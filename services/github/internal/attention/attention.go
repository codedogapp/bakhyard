// Package attention serves github's side of the attention contract: the pull requests that wait
// for the user's review and the open issues assigned to the user, as items that need the user on
// the Home (docs/plan.md decision 208).
package attention

import (
	"context"
	"fmt"

	"connectrpc.com/connect/v2"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// reason says why an item needs the user, in the words that end its detail.
type reason string

const (
	reviewRequested reason = "Review requested"
	assigned        reason = "Assigned"
)

// Lister lists the pull requests that wait for the user's review and the open issues assigned to
// the user as the attention contract's items.
type Lister struct {
	reviews *github.Poller
	issues  *github.Poller
}

// NewLister returns a lister of the review requests that reviews finds on GitHub and of the
// assigned issues that issues finds there. It lists none when GitHub isn't set up and both are
// nil.
func NewLister(reviews, issues *github.Poller) *Lister {
	return &Lister{reviews: reviews, issues: issues}
}

// ListItems lists the review requests, then the assigned issues, each in the order of the latest
// search of its list to succeed, once the first search of each has ended, or returns ctx's error
// if ctx ends first. Until both searches have succeeded once, the items are unavailable.
func (l *Lister) ListItems(
	ctx context.Context,
	_ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	reviews, err := items(ctx, l.reviews, reviewRequested)
	if err != nil {
		return nil, err
	}

	issues, err := items(ctx, l.issues, assigned)
	if err != nil {
		return nil, err
	}

	return attentionv1.ListItemsResponse_builder{Items: append(reviews, issues...)}.Build(), nil
}

// items returns the attention items of what the latest search of poller to succeed found, which
// need the user for why, or none without poller.
func items(
	ctx context.Context, poller *github.Poller, why reason,
) ([]*attentionv1.Item, error) {
	if poller == nil {
		return nil, nil
	}

	latest, err := poller.Latest(ctx)
	if err != nil {
		return nil, err
	}

	if latest.Searched.IsZero() {
		return nil, connect.NewError(connect.CodeUnavailable,
			"GitHub's searches haven't all succeeded yet")
	}

	listed := make([]*attentionv1.Item, 0, len(latest.Found.Items))
	for _, found := range latest.Found.Items {
		listed = append(listed, item(found, why))
	}

	return listed, nil
}

// item returns the attention item of found, which needs the user for why: its title, over its
// repository and number, why, and Draft for a draft pull request.
func item(found github.Item, why reason) *attentionv1.Item {
	detail := fmt.Sprintf("%s #%d · %s", found.Repo, found.ID, why)
	if found.Draft {
		detail += " · Draft"
	}

	return attentionv1.Item_builder{
		Id:     new(fmt.Sprintf("%s#%d", found.Repo, found.ID)),
		Title:  new(found.Title),
		Detail: new(detail),
		Source: new(attentionv1.Source_SOURCE_GITHUB),
	}.Build()
}
