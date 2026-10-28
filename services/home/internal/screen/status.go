package screen

// State says how asking one of home's peers what needs the user went.
type State string

// The states that a source can be in.
const (
	// StateOK means the peer answered, or doesn't serve the contract and so has nothing to list.
	StateOK State = "ok"
	// StateUnavailable means the peer failed or didn't answer in time: its items are missing.
	StateUnavailable State = "unavailable"
)

// SourceStatus is how asking the peer that App names went.
type SourceStatus struct {
	App   App   `json:"app"`
	State State `json:"state"`
}
