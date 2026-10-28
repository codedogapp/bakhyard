// Package screen builds what the Home screen shows: who it greets and the items that need the
// user, most urgent first.
package screen

import (
	"cmp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// defaultName stands in for a blank name, as in "Good evening, there."
const defaultName = "there"

// App is the Bakhyard app an item belongs to, which is the service that listed it.
type App string

// Source is the service an item comes from: an outside one, or Bakhyard itself.
type Source string

// The sources of the items so far.
const (
	SourceBakhyard  Source = "Bakhyard"
	SourceBitbucket Source = "Bitbucket"
	SourceJira      Source = "Jira"
)

// Item is one thing that needs the user. Its ID is "<app>/<id>", and a zero Due means no due
// time.
type Item struct {
	ID     string    `json:"id"`
	App    App       `json:"app"`
	Title  string    `json:"title"`
	Detail string    `json:"detail,omitempty"`
	Source Source    `json:"source"`
	Due    time.Time `json:"due,omitzero"`
	Late   bool      `json:"late"`
}

// Home is everything the Home screen shows. Sources says how asking each of home's peers went,
// so that a Home without some of its items doesn't look whole. It has no week until the time
// service logs hours (docs/plan.md decision 161).
type Home struct {
	Name    string         `json:"name"`
	Items   []Item         `json:"items"`
	Sources []SourceStatus `json:"sources"`
}

// New builds the Home at now for the user called name (see cleanName), with the items and the
// sources that its peers gave. It marks the items past their due time as late and puts the most
// urgent first. The caller's items are left as they are.
func New(name string, items []Item, sources []SourceStatus, now time.Time) Home {
	sorted := slices.Clone(items)
	for i := range sorted {
		sorted[i].Late = !sorted[i].Due.IsZero() && sorted[i].Due.Before(now)
	}

	slices.SortFunc(sorted, byUrgency)

	return Home{Name: cleanName(name), Items: sorted, Sources: sources}
}

// cleanName drops the whitespace around name and replaces invalid UTF-8, which JSON can't encode,
// with U+FFFD. A name left empty becomes "there".
func cleanName(name string) string {
	valid := strings.ToValidUTF8(name, string(utf8.RuneError))

	return cmp.Or(strings.TrimSpace(valid), defaultName)
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
