// Package screen builds what the Home screen shows: the items that need the user, most urgent
// first, and the hours logged this week.
package screen

import (
	"cmp"
	"slices"
	"time"
)

// App is the Bakhyard app an item belongs to.
type App string

// The apps that have items so far.
const (
	AppGitHub  App = "github"
	AppPlanner App = "planner"
	AppPulls   App = "pulls"
	AppTime    App = "time"
	AppTodos   App = "todos"
)

// Source is the outside service an item comes from.
type Source string

// The sources of the items so far.
const (
	SourceBitbucket Source = "Bitbucket"
	SourceCopilot   Source = "Copilot"
	SourceGitHub    Source = "GitHub"
	SourceJira      Source = "Jira"
)

// Item is one thing that needs the user. Its ID is "<app>/<id>", and a zero Due means no due
// time.
type Item struct {
	ID     string
	App    App
	Title  string
	Detail string
	Source Source
	Due    time.Time
	Late   bool
}

// Week is the hours logged this week against the target.
type Week struct {
	LoggedHours float64
	TargetHours float64
}

// Home is everything the Home screen shows.
type Home struct {
	Items []Item
	Week  Week
}

// New builds the Home at now. It marks the items past their due time as late and puts the most
// urgent first. The caller's items are left as they are.
func New(items []Item, week Week, now time.Time) Home {
	sorted := slices.Clone(items)
	for i := range sorted {
		sorted[i].Late = !sorted[i].Due.IsZero() && sorted[i].Due.Before(now)
	}

	slices.SortFunc(sorted, byUrgency)

	return Home{Items: sorted, Week: week}
}

// byUrgency orders by due time, then title, then ID. Late items come first because their due
// times are the earliest.
func byUrgency(a, b Item) int {
	return cmp.Or(
		compareDue(a.Due, b.Due),
		cmp.Compare(a.Title, b.Title),
		cmp.Compare(a.ID, b.ID),
	)
}

// compareDue puts earlier due times first and items with no due time last.
func compareDue(a, b time.Time) int {
	switch {
	case a.IsZero() == b.IsZero():
		return a.Compare(b)
	case a.IsZero():
		return 1
	default:
		return -1
	}
}
