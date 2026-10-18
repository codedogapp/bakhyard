// Package demo holds the sample Home data, served until the apps have real data.
package demo

import (
	"time"

	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

const (
	nudgeOverdue   = 2 * time.Hour
	ticketOverdue  = 24 * time.Hour
	hoursDueIn     = 3 * time.Hour
	tokenExpiresIn = 5 * 24 * time.Hour
	loggedHours    = 31.5
	targetHours    = 40
)

// Items returns the sample items, with due times relative to now.
func Items(now time.Time) []screen.Item {
	return []screen.Item{
		{
			ID:     "pulls/PAY-1423",
			App:    screen.AppPulls,
			Title:  "Nudge reviewers on PAY-1423",
			Source: screen.SourceBitbucket,
			Due:    now.Add(-nudgeOverdue),
		},
		{
			ID:     "todos/PAY-1398",
			App:    screen.AppTodos,
			Title:  "PAY-1398 Settlement report export",
			Detail: "Was due yesterday.",
			Source: screen.SourceJira,
			Due:    now.Add(-ticketOverdue),
		},
		{
			ID:     "time/today",
			App:    screen.AppTime,
			Title:  "Log today's hours",
			Source: screen.SourceJira,
			Due:    now.Add(hoursDueIn),
		},
		{
			ID:     "time/token",
			App:    screen.AppTime,
			Title:  "Jira token expires in 5 days",
			Source: screen.SourceJira,
			Due:    now.Add(tokenExpiresIn),
		},
		{
			ID:     "github/dotfiles-14",
			App:    screen.AppGitHub,
			Title:  "Review requested: dotfiles#14",
			Source: screen.SourceGitHub,
		},
		{
			ID:     "planner/home-lab-backups",
			App:    screen.AppPlanner,
			Title:  "Finish plan: home lab backups",
			Source: screen.SourceCopilot,
		},
	}
}

// Week returns the sample hours logged this week.
func Week() screen.Week {
	return screen.Week{LoggedHours: loggedHours, TargetHours: targetHours}
}
