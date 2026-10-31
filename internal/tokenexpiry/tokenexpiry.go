// Package tokenexpiry reminds the user to renew an API token before it expires, with an item in
// the Home's "Needs you today" (docs/plan.md decision 500).
package tokenexpiry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
)

// lead is how long before a token expires its reminder starts.
const lead = 14 * 24 * time.Hour

// Token is an API token that expires.
type Token struct {
	// Name names the token, as in "Renew the Jira token".
	Name string
	// Source is the outside service that the token is for.
	Source attentionv1.Source
	// Expires tells when the token expires.
	Expires Expiry
}

// Expiry returns when a token expires, or the zero time for a token that never expires, or whose
// expiry the service doesn't know yet.
type Expiry func(context.Context) (time.Time, error)

// On returns the expiry of a token that expires at expires, such as the date that FromEnv reads.
func On(expires time.Time) Expiry {
	return func(context.Context) (time.Time, error) { return expires, nil }
}

// FromEnv returns the date that the setting name holds as YYYY-MM-DD, at its start in UTC, since
// Atlassian doesn't say at what time of the day a token stops. It returns the zero time when the
// setting is unset.
func FromEnv(getenv func(string) string, name string) (time.Time, error) {
	value := getenv(name)
	if value == "" {
		return time.Time{}, nil
	}

	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %q is not a date, YYYY-MM-DD", name, value)
	}

	return date, nil
}

// item returns the item that reminds the user at now to renew the token, which expires at
// expires, due then, so that it shows late after. Its ID is the token's name in lower case, then
// -token, such as jira-token. It returns nil when expires is zero, or more than lead after now.
func (t Token) item(expires, now time.Time) *attentionv1.Item {
	if expires.IsZero() || now.Before(expires.Add(-lead)) {
		return nil
	}

	return attentionv1.Item_builder{
		Id:     new(strings.ToLower(t.Name) + "-token"),
		Title:  new("Renew the " + t.Name + " token"),
		Source: new(t.Source),
		Due:    timestamppb.New(expires),
	}.Build()
}
