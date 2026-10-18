// Package api serves the home API that api/home.yaml describes. The spec's types and server
// are generated into api.gen.go by `mise run generate`.
package api

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/home/internal/demo"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// Register serves the home API on router; other methods get 405. Due times are relative to
// now().
func Register(router chi.Router, now func() time.Time) {
	HandlerFromMux(NewStrictHandler(server{now: now}, nil), router)
}

// server implements the home API's operations.
type server struct {
	now func() time.Time
}

// GetHome returns the Home screen built from the demo data.
func (s server) GetHome(context.Context, GetHomeRequestObject) (GetHomeResponseObject, error) {
	at := s.now()

	return GetHome200JSONResponse(toHome(screen.New(demo.Items(at), demo.Week(), at))), nil
}

// toHome converts the domain's Home to the API's. Week converts directly, so a field added to
// one side and not the other fails to compile.
func toHome(home screen.Home) Home {
	items := make([]Item, 0, len(home.Items))
	for _, item := range home.Items {
		items = append(items, toItem(item))
	}

	return Home{Items: items, Week: Week(home.Week)}
}

// toItem converts one of the domain's items to the API's. The apps and sources keep their
// values; the tests check that the spec allows them.
func toItem(item screen.Item) Item {
	return Item{
		ID:     item.ID,
		App:    App(item.App),
		Title:  item.Title,
		Detail: item.Detail,
		Source: Source(item.Source),
		Due:    item.Due,
		Late:   item.Late,
	}
}
