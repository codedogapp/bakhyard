package main

import (
	"net/http"
	"slices"
	"testing"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
)

func TestTimeListsTheWeekdaysOfItsReportWithNothingLogged(t *testing.T) {
	router, err := routes(t.Context(), envOf(jiraSettings(jiraServer(t))),
		secretsWith(map[string]string{"jira-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	got := listItems(t, router)

	// Jira's worklog started a minute ago, which may be yesterday: the report says which day has it.
	var want []string

	for _, day := range reportOf(t, router).Days {
		date, err := time.ParseInLocation(time.DateOnly, day.Date, time.Local)
		if err != nil {
			t.Fatal(err)
		}

		if day.LoggedHours == 0 && date.Weekday() != time.Saturday &&
			date.Weekday() != time.Sunday {
			due := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.Local)
			want = append(want, day.Date+" due "+due.Format(time.RFC3339))
		}
	}

	if len(want) == 0 || !slices.Equal(got, want) {
		t.Errorf("got %q, want %q: a day each, due at 17:00", got, want)
	}
}

func TestTimeListsNoAttentionItemsWithoutJira(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil))
	if err != nil {
		t.Fatal(err)
	}

	if got := listItems(t, router); len(got) != 0 {
		t.Errorf("got %q, want no items", got)
	}
}

// listItems returns each item that router lists over the attention contract, called with the gRPC
// protocol, as "<id> due <its due time in the local zone>".
func listItems(t *testing.T, router http.Handler) []string {
	t.Helper()

	client := attentionv1connect.NewAttentionServiceClient(grpcClient(t, router))

	res, err := client.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var items []string

	for _, item := range res.GetItems() {
		due := item.GetDue().AsTime().In(time.Local)
		items = append(items, item.GetId()+" due "+due.Format(time.RFC3339))
	}

	return items
}
