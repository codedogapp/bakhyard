// Package attention serves pulls' side of the attention contract: the user's open pull requests,
// as the items that need the user on the Home (decision 158).
package attention

import (
	"context"
	"fmt"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// Lister lists the user's open pull requests as the attention contract's items.
type Lister struct {
	mine *bitbucket.Poller
}

// NewLister returns a lister of the pull requests that mine has found on Bitbucket, which lists
// none when Bitbucket isn't set up and mine is nil.
func NewLister(mine *bitbucket.Poller) *Lister {
	return &Lister{mine: mine}
}

// ListItems lists the pull requests that the latest poll to succeed found, in their order, once
// the first poll has ended, or returns ctx's error if ctx ends first.
func (l *Lister) ListItems(
	ctx context.Context,
	_ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	if l.mine == nil {
		return attentionv1.ListItemsResponse_builder{}.Build(), nil
	}

	latest, err := l.mine.Latest(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*attentionv1.Item, 0, len(latest.Found.Pulls))
	for _, pull := range latest.Found.Pulls {
		items = append(items, item(pull))
	}

	return attentionv1.ListItemsResponse_builder{Items: items}.Build(), nil
}

// item returns the attention item of pull: its title, over its repository and number, and Draft
// for a draft.
func item(pull bitbucket.Pull) *attentionv1.Item {
	detail := fmt.Sprintf("%s #%d", pull.Repo, pull.ID)
	if pull.Draft {
		detail += " · Draft"
	}

	return attentionv1.Item_builder{
		Id:     new(fmt.Sprintf("%s#%d", pull.Repo, pull.ID)),
		Title:  new(pull.Title),
		Detail: new(detail),
		Source: new(attentionv1.Source_SOURCE_BITBUCKET),
	}.Build()
}
