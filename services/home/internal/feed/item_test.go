package feed_test

import (
	"slices"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

func TestGatherNamesCopilotAsTheSourceOfPlannersReminder(t *testing.T) {
	expires := time.Date(2026, time.October, 27, 22, 0, 0, 0, time.UTC)
	planner := serveAttention(t, answer(attentionv1.Item_builder{
		Id: new("copilot-token"), Title: new("Renew the Copilot token"),
		Source: new(attentionv1.Source_SOURCE_COPILOT), Due: timestamppb.New(expires),
	}.Build()))

	got, sources, logs := gather(t, env.Peer{Name: "planner", Address: planner})

	want := []screen.Item{{
		ID: "planner/copilot-token", App: "planner", Title: "Renew the Copilot token",
		Source: screen.SourceCopilot, Due: expires,
	}}
	if !slices.EqualFunc(got, want, sameItem) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	checkSources(t, sources, ok("planner"))

	if logs != "" {
		t.Errorf("logged %q, want nothing", logs)
	}
}
