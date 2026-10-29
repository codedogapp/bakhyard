package jira

import "regexp"

// keyRule matches a Jira ticket's key, as Jira makes them: its project's key, an upper-case
// letter and at least one more upper-case letter, digit or underscore, then a hyphen and the
// ticket's number (decision 125).
var keyRule = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-[1-9][0-9]*$`)

// IsKey reports whether s is a Jira ticket's key, such as ABC-123.
func IsKey(s string) bool {
	return keyRule.MatchString(s)
}
