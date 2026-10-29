package main

import (
	"net/http"
	"testing"

	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
)

func TestTimeAnswersTheWeekOfItsReport(t *testing.T) {
	router, err := routes(t.Context(), envOf(jiraSettings(jiraServer(t))),
		secretsWith(map[string]string{"jira-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	got := getWeek(t, router)
	want := reportOf(t, router)

	// Jira logs an hour more with each read, so the two agree only when they come from the same
	// read. Its worklog started a minute ago, which may be last week: the report says which.
	if !got.HasWeek() || got.GetWeek().GetLoggedHours() != want.Week.LoggedHours ||
		got.GetWeek().GetTargetHours() != want.Week.TargetHours || want.Week.TargetHours != 40 {
		t.Errorf("got %v, want the report's week %+v, with a target of 40 hours", got, want.Week)
	}
}

func TestTimeAnswersNoWeekWithoutJira(t *testing.T) {
	router, err := routes(t.Context(), envOf(nil), secretsWith(nil))
	if err != nil {
		t.Fatal(err)
	}

	if got := getWeek(t, router); got.HasWeek() {
		t.Errorf("got %v, want no week", got)
	}
}

// getWeek returns router's answer to GetWeek over the time contract, called with the gRPC protocol.
func getWeek(t *testing.T, router http.Handler) *timev1.GetWeekResponse {
	t.Helper()

	client := timev1connect.NewTimeServiceClient(grpcClient(t, router))

	res, err := client.GetWeek(t.Context(), &timev1.GetWeekRequest{})
	if err != nil {
		t.Fatal(err)
	}

	return res
}
