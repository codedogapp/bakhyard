package todo

import "errors"

// Priority is a todo's priority (decision 102); a new todo is Normal.
type Priority string

// The three priorities, named as the database and the API name them (decision 115).
const (
	High   Priority = "high"
	Normal Priority = "normal"
	Low    Priority = "low"
)

// ErrPriority is the error of a priority that isn't High, Normal or Low.
var ErrPriority = errors.New("invalid priority: not high, normal or low")

// valid reports whether p is High, Normal or Low.
func (p Priority) valid() bool {
	return p == High || p == Normal || p == Low
}
