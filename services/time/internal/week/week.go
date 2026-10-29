// Package week serves time's side of the time contract: this week's hours against the week's
// target, for the Home's "This week" (docs/plan.md decision 185).
package week

import (
	"context"

	"connectrpc.com/connect/v2"

	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// Server answers the week of the time reports that a poller keeps.
type Server struct {
	reports *worklogs.Poller
}

// NewServer returns a server of the week in the reports that reports keeps, which answers no week
// when Jira isn't set up and reports is nil.
func NewServer(reports *worklogs.Poller) *Server {
	return &Server{reports: reports}
}

// GetWeek answers the week of the report that the latest poll to succeed worked out, once the
// first poll has ended, or returns ctx's error if ctx ends first. Until a poll succeeds, the week
// is unavailable.
func (s *Server) GetWeek(
	ctx context.Context,
	_ *timev1.GetWeekRequest,
) (*timev1.GetWeekResponse, error) {
	if s.reports == nil {
		return timev1.GetWeekResponse_builder{}.Build(), nil
	}

	latest, err := s.reports.Latest(ctx)
	if err != nil {
		return nil, err
	}

	if latest.Searched.IsZero() {
		return nil, connect.NewError(connect.CodeUnavailable,
			"no read of the worklogs on Jira has succeeded yet")
	}

	week := latest.Found.Week

	return timev1.GetWeekResponse_builder{Week: timev1.Week_builder{
		LoggedHours: new(week.LoggedHours),
		TargetHours: new(week.TargetHours),
	}.Build()}.Build(), nil
}
