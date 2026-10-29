package screen

// State says how asking one of home's peers went: any of them for its items, or time for the week.
type State string

// The states that a source, or the week, can be in.
const (
	// StateOK means the peer answered, or doesn't serve the attention contract and so has nothing
	// to list.
	StateOK State = "ok"
	// StateUnavailable means the peer failed or didn't answer in time: its items, or the week, are
	// missing.
	StateUnavailable State = "unavailable"
)

// SourceStatus is how asking the peer that App names went.
type SourceStatus struct {
	App   App   `json:"app"`
	State State `json:"state"`
}
