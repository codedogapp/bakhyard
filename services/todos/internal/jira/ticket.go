package jira

// Category is what a ticket's status stands for. Each workflow names its statuses as it likes,
// but every status falls into one of Jira's few status categories.
type Category string

// Jira's status categories, under names of the app's own.
const (
	ToDo       Category = "to-do"
	InProgress Category = "in-progress"
	Done       Category = "done"
	// Unknown stands for Jira's "No Category" and for any category newer than this client.
	Unknown Category = "unknown"
)

// categoryOf returns the Category of the status category whose key Jira sends.
func categoryOf(key string) Category {
	switch key {
	case "new":
		return ToDo
	case "indeterminate":
		return InProgress
	case "done":
		return Done
	default:
		return Unknown
	}
}

// Ticket is one of the user's Jira tickets.
type Ticket struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
	// Status is the name of the ticket's status, as its workflow names it.
	Status   string   `json:"status"`
	Category Category `json:"category"`
	// URL is the ticket's page on the Jira site.
	URL string `json:"url"`
}
