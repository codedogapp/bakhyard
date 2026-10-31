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
// from 14 days before it expires.
func NewLister(items attentionv1connect.AttentionServiceHandler, token Token) *Lister {
	return &Lister{items: items, token: token}
}

// ListItems lists the items that the service's lister lists, then the token's reminder when it's
// due within 14 days or already past. When the service's lister fails, it fails with its error.
func (l *Lister) ListItems(
	ctx context.Context,
	req *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	res, err := l.items.ListItems(ctx, req)
	if err != nil {
		return nil, err
	}

	if reminder := l.token.item(time.Now()); reminder != nil {
		res.SetItems(append(res.GetItems(), reminder))
	}

	return res, nil
}
