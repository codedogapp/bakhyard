package feed

import (
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// homeItem is the Home's item for one that the peer called app listed: its ID is "<app>/<id>",
// and it has a due time only if the peer sent one.
func homeItem(app string, listed *attentionv1.Item) screen.Item {
	item := screen.Item{
		ID:     app + "/" + listed.GetId(),
		App:    screen.App(app),
		Title:  listed.GetTitle(),
		Detail: listed.GetDetail(),
		Source: source(listed.GetSource()),
	}

	if listed.HasDue() {
		item.Due = listed.GetDue().AsTime()
	}

	return item
}

// source names where an item comes from. An item without an outside source comes from Bakhyard
// itself, and one whose source home doesn't know yet gets the same name.
func source(source attentionv1.Source) screen.Source {
	switch source {
	case attentionv1.Source_SOURCE_BITBUCKET:
		return screen.SourceBitbucket
	case attentionv1.Source_SOURCE_JIRA:
		return screen.SourceJira
	default:
		return screen.SourceBakhyard
	}
}
