// Package attention serves notify's side of the attention contract: the laptop's alerts that
// fire, as the items that need the user on the Home (decision 606).
package attention

import (
	"cmp"
	"context"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
)

// Lister lists the alerts that fire as the attention contract's items.
type Lister struct {
	alerts func(context.Context) ([]history.Entry, error)
}

// NewLister returns a lister of the alerts that fire among those that alerts lists.
func NewLister(alerts func(context.Context) ([]history.Entry, error)) *Lister {
	return &Lister{alerts: alerts}
}

// ListItems lists an item for each alert that fires, in the order of the alerts' list.
func (l *Lister) ListItems(
	ctx context.Context,
	_ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	entries, err := l.alerts(ctx)
	if err != nil {
		return nil, err
	}

	var items []*attentionv1.Item

	for _, entry := range entries {
		if entry.ResolvedAt.IsZero() {
			items = append(items, item(entry))
		}
	}

	return attentionv1.ListItemsResponse_builder{Items: items}.Build(), nil
}

// item returns the attention item of entry, an alert that fires: its summary, or else its name,
// over its description, with no source, so that it comes from Bakhyard, and no due time. Its ID
// is the alert's fingerprint and when it started, which tell each time that an alert fired apart.
func item(entry history.Entry) *attentionv1.Item {
	return attentionv1.Item_builder{
		Id:     new(entry.Fingerprint + "@" + entry.StartedAt.UTC().Format(time.RFC3339Nano)),
		Title:  new(cmp.Or(entry.Summary, entry.Name)),
		Detail: new(entry.Description),
	}.Build()
}
