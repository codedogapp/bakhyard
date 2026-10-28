package attention

import (
	"context"
	"slices"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// inbox returns the items of the Jira inbox's tickets, in the order of open, the user's open
// tickets: those that no todo names, a done one included, and that the user hasn't dismissed, the
// Todos page's rule (decisions 128 and 160). It reads the dismissed tickets only when there are
// open tickets, and leaves open as it is, since the poller shares it.
func (l *Lister) inbox(
	ctx context.Context, open []jira.Ticket, todos []todo.Todo,
) ([]*attentionv1.Item, error) {
	if len(open) == 0 {
		return nil, nil
	}

	dismissed, err := l.dismissed(ctx)
	if err != nil {
		return nil, err
	}

	var items []*attentionv1.Item

	for _, ticket := range open {
		named := slices.ContainsFunc(todos, func(listed todo.Todo) bool {
			return listed.Ticket == ticket.Key
		})
		if !named && !slices.Contains(dismissed, ticket.Key) {
			items = append(items, inboxItem(ticket))
		}
	}

	return items, nil
}

// inboxItem returns the attention item of a ticket in the Jira inbox: its key as the ID, its
// summary as the title, and why it needs the user after its key. Its source is Jira.
func inboxItem(ticket jira.Ticket) *attentionv1.Item {
	return attentionv1.Item_builder{
		Id:     new(ticket.Key),
		Title:  new(ticket.Summary),
		Detail: new(ticket.Key + " · In the Jira inbox"),
		Source: new(attentionv1.Source_SOURCE_JIRA),
	}.Build()
}
