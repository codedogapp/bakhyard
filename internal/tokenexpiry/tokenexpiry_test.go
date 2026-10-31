package tokenexpiry_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
)

// expiresVar is the setting of the tests' token's date.
const expiresVar = "BAKHYARD_JIRA_TOKEN_EXPIRES"

// fixTheLogin is the item that the tests' service lists, and loginLine its line, as lines writes
// it.
var fixTheLogin = attentionv1.Item_builder{
	Id:     new("WEB-9"),
	Title:  new("Fix the login"),
	Detail: new("WEB-9 · Done in Jira"),
	Source: new(attentionv1.Source_SOURCE_JIRA),
}.Build()

const loginLine = "WEB-9 | Fix the login | WEB-9 · Done in Jira | SOURCE_JIRA | no due"

func TestFromEnvReadsTheDateAtItsStartInUTC(t *testing.T) {
	got, err := tokenexpiry.FromEnv(envOf("2027-10-31"), expiresVar)

	want := time.Date(2027, time.October, 31, 0, 0, 0, 0, time.UTC)
	if err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v and %v, want %v", got, err, want)
	}
}

func TestFromEnvReadsNoDateWhenUnset(t *testing.T) {
	got, err := tokenexpiry.FromEnv(envOf(""), expiresVar)
	if err != nil || !got.IsZero() {
		t.Errorf("got %v and %v, want no date", got, err)
	}
}

func TestFromEnvRejectsWhatIsNotADate(t *testing.T) {
	for _, value := range []string{
		"31/10/2027", "2027-10-32", "2027-02-29", "2027-1-5", " 2027-10-31", "2027-10-31 ",
		"2027-10-31T00:00:00Z", "soon",
	} {
		got, err := tokenexpiry.FromEnv(envOf(value), expiresVar)

		want := fmt.Sprintf("%s: %q is not a date, YYYY-MM-DD", expiresVar, value)
		if err == nil || err.Error() != want || !got.IsZero() {
			t.Errorf("%q: got %v and %v, want the error %s", value, got, err, want)
		}
	}
}

func TestListItemsRemindsFrom14DaysBeforeTheTokenExpires(t *testing.T) {
	for _, test := range []struct {
		name     string
		expires  time.Duration
		reminded bool
	}{
		{"more than 14 days ahead", 14*24*time.Hour + time.Nanosecond, false},
		{"14 days ahead", 14 * 24 * time.Hour, true},
		{"an hour ahead", time.Hour, true},
		{"now", 0, true},
		{"a day ago", -24 * time.Hour, true},
	} {
		synctest.Test(t, func(t *testing.T) {
			expires := time.Now().Add(test.expires)
			lister := tokenexpiry.NewLister(listing(fixTheLogin), jiraToken(expires))

			want := []string{loginLine}
			if test.reminded {
				want = append(want, "jira-token | Renew the Jira token |  | SOURCE_JIRA | "+
					expires.UTC().String())
			}

			if got := list(t, lister); !slices.Equal(got, want) {
				t.Errorf("%s: got %q, want %q", test.name, got, want)
			}
		})
	}
}

func TestListItemsRemindsWhenTheServiceListsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expires := time.Now().Add(time.Hour)
		lister := tokenexpiry.NewLister(listing(), jiraToken(expires))

		want := []string{"jira-token | Renew the Jira token |  | SOURCE_JIRA | " +
			expires.UTC().String()}
		if got := list(t, lister); !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestListItemsNeverRemindsOfATokenThatNeverExpires(t *testing.T) {
	lister := tokenexpiry.NewLister(listing(fixTheLogin), jiraToken(time.Time{}))

	if got := list(t, lister); !slices.Equal(got, []string{loginLine}) {
		t.Errorf("got %q, want only the service's item", got)
	}
}

func TestListItemsFailsWhenTheServiceFails(t *testing.T) {
	down := errors.New("the database is down")
	failing := listerFunc(func(context.Context, *attentionv1.ListItemsRequest) (
		*attentionv1.ListItemsResponse, error,
	) {
		return nil, down
	})
	lister := tokenexpiry.NewLister(failing, jiraToken(time.Now()))

	res, err := lister.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if !errors.Is(err, down) || res != nil {
		t.Errorf("got %v and %v, want no answer and the service's error", res, err)
	}
}

func TestListItemsFailsWhenTheExpiryFails(t *testing.T) {
	unknown := errors.New("the expiry is unknown")
	token := jiraToken(time.Time{})
	token.Expires = func(context.Context) (time.Time, error) { return time.Time{}, unknown }

	res, err := tokenexpiry.NewLister(listing(fixTheLogin), token).ListItems(t.Context(),
		&attentionv1.ListItemsRequest{})
	if !errors.Is(err, unknown) || res != nil {
		t.Errorf("got %v and %v, want no answer and the expiry's error", res, err)
	}
}

// call is the key of a value that marks a call's context.
type call struct{}

func TestListItemsListsTheServicesItemsAndTheExpiryWithinTheCall(t *testing.T) {
	asked := &attentionv1.ListItemsRequest{}
	within := listerFunc(func(ctx context.Context, req *attentionv1.ListItemsRequest) (
		*attentionv1.ListItemsResponse, error,
	) {
		if ctx.Value(call{}) == nil || req != asked {
			t.Error("listed the service's items outside the call")
		}

		return &attentionv1.ListItemsResponse{}, nil
	})
	token := jiraToken(time.Time{})
	token.Expires = func(ctx context.Context) (time.Time, error) {
		if ctx.Value(call{}) == nil {
			t.Error("asked when the token expires outside the call")
		}

		return time.Time{}, nil
	}

	_, err := tokenexpiry.NewLister(within, token).ListItems(
		context.WithValue(t.Context(), call{}, true), asked)
	if err != nil {
		t.Fatal(err)
	}
}

// listerFunc is a service's lister of items, as a function.
type listerFunc func(context.Context, *attentionv1.ListItemsRequest) (
	*attentionv1.ListItemsResponse, error)

func (f listerFunc) ListItems(
	ctx context.Context,
	req *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	return f(ctx, req)
}

// listing returns a service's lister that lists items.
func listing(items ...*attentionv1.Item) listerFunc {
	return func(context.Context, *attentionv1.ListItemsRequest) (
		*attentionv1.ListItemsResponse, error,
	) {
		return attentionv1.ListItemsResponse_builder{Items: items}.Build(), nil
	}
}

// jiraToken returns the Jira token, which expires at expires, or never when it's zero.
func jiraToken(expires time.Time) tokenexpiry.Token {
	return tokenexpiry.Token{
		Name: "Jira", Source: attentionv1.Source_SOURCE_JIRA, Expires: tokenexpiry.On(expires),
	}
}

// envOf returns an environment whose only setting is the tests' token's date, set to value.
func envOf(value string) func(string) string {
	return func(name string) string { return map[string]string{expiresVar: value}[name] }
}

// list returns the items that lister lists, one line each: their ID, title, detail, source and
// due time.
func list(t *testing.T, lister *tokenexpiry.Lister) []string {
	t.Helper()

	res, err := lister.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var lines []string

	for _, item := range res.GetItems() {
		due := "no due"
		if item.HasDue() {
			due = item.GetDue().AsTime().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", item.GetId(),
			item.GetTitle(), item.GetDetail(), item.GetSource(), due))
	}

	return lines
}
