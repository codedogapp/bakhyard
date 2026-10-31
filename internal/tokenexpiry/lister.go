package tokenexpiry

import (
	"context"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
)

// Lister lists a service's items over the attention contract, then the reminder to renew the
// token that the service uses.
type Lister struct {
	items attentionv1connect.AttentionServiceHandler
	token Token
}

// NewLister returns a lister of the items that items lists, and of the reminder to renew token
// from 14 days before it expires. A service with no items of its own, such as planner, gives nil
// items, and lists the reminder alone.
func NewLister(items attentionv1connect.AttentionServiceHandler, token Token) *Lister {
	return &Lister{items: items, token: token}
}

// ListItems lists the items that the service's lister lists, then the token's reminder when it's
// due within 14 days or already past. When the service's lister fails, or the token's expiry, it
// fails with its error.
func (l *Lister) ListItems(
	ctx context.Context,
	req *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	res, err := l.listed(ctx, req)
	if err != nil {
		return nil, err
	}

	expires, err := l.token.Expires(ctx)
	if err != nil {
		return nil, err
	}

	if reminder := l.token.item(expires, time.Now()); reminder != nil {
		res.SetItems(append(res.GetItems(), reminder))
	}

	return res, nil
}

// listed returns the items that the service's lister lists, or none without one.
func (l *Lister) listed(
	ctx context.Context,
	req *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	if l.items == nil {
		return &attentionv1.ListItemsResponse{}, nil
	}

	return l.items.ListItems(ctx, req)
}
