// Package attention serves todos' side of the attention contract: the Jira inbox's tickets and the
// todos left to do that need the user, as the items on the Home (decisions 159 and 160).
package attention

import (
	"context"
	"slices"
	"strconv"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// Lister lists the Jira inbox's tickets and the todos that need the user as the attention
// contract's items.
type Lister struct {
	list      func(context.Context) ([]todo.Todo, error)
	dismissed func(context.Context) ([]string, error)
	tickets   *jira.Poller
}

// NewLister returns a lister of the todos that list reads, such as a Store's List, and of the Jira
// inbox, which leaves out the tickets that dismissed reads, such as a Store's Dismissed. It learns
// from tickets which tickets are open and which are done; tickets is nil when Jira isn't set up.
func NewLister(
	list func(context.Context) ([]todo.Todo, error),
	dismissed func(context.Context) ([]string, error),
	tickets *jira.Poller,
) *Lister {
	return &Lister{list: list, dismissed: dismissed, tickets: tickets}
}

// ListItems lists the Jira inbox's tickets, then, in the list's order, the todos left to do that
// are High or whose ticket the latest Jira poll to succeed found done. With Jira set up, it waits
// for the first poll to end, and returns ctx's error if ctx ends first.
func (l *Lister) ListItems(
	ctx context.Context,
	_ *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	found, err := l.found(ctx)
	if err != nil {
		return nil, err
	}

	todos, err := l.list(ctx)
	if err != nil {
		return nil, err
	}

	items, err := l.inbox(ctx, found.Tickets, todos)
	if err != nil {
		return nil, err
	}

	for _, listed := range todos {
		ticketDone := slices.ContainsFunc(found.Done, func(ticket jira.Ticket) bool {
			return ticket.Key == listed.Ticket
		})
		if !listed.Done && (listed.Priority == todo.High || ticketDone) {
			items = append(items, todoItem(listed, ticketDone))
		}
	}

	return attentionv1.ListItemsResponse_builder{Items: items}.Build(), nil
}

// found returns what the latest Jira poll to succeed found, once the first poll has ended, and
// nothing when Jira isn't set up.
func (l *Lister) found(ctx context.Context) (jira.Found, error) {
	if l.tickets == nil {
		return jira.Found{}, nil
	}

	latest, err := l.tickets.Latest(ctx)
	if err != nil {
		return jira.Found{}, err
	}

	return latest.Found, nil
}

// todoItem returns the attention item of a todo left to do: its ID and title, and why it needs
// the user, after the key of its ticket when it names one. Its source is Jira when it names a
// ticket, and unset when it doesn't.
func todoItem(listed todo.Todo, ticketDone bool) *attentionv1.Item {
	detail := "High priority"
	if ticketDone {
		detail = "Done in Jira"
	}

	var source *attentionv1.Source

	if listed.Ticket != "" {
		detail = listed.Ticket + " · " + detail
		source = new(attentionv1.Source_SOURCE_JIRA)
	}

	return attentionv1.Item_builder{
		Id:     new(strconv.FormatInt(listed.ID, 10)),
		Title:  new(listed.Title),
		Detail: new(detail),
		Source: source,
	}.Build()
}
