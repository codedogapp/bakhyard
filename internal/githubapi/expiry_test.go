package githubapi_test

import (
	"net/http"
	"testing"
	"time"
)

// expirationHeader is the header in which GitHub tells when a token expires.
const expirationHeader = "GitHub-Authentication-Token-Expiration"

// expiringAt returns a stand-in GitHub's OK answer that says the token expires at value.
func expiringAt(value string) answer {
	return answer{http.StatusOK, http.Header{expirationHeader: {value}}}
}

func TestExpiryAsksGitHubAboutTheUser(t *testing.T) {
	client, notes := standIn(t, "", expiringAt("2027-10-27 22:00:00 UTC"))

	got, err := client.Expiry(t.Context())

	want := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)
	if err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v and %v, want %v", got, err, want)
	}

	request := asked{
		"GET /user", "Bearer secret", "application/vnd.github+json", "2026-03-10", "bakhyard",
	}
	if note := <-notes; note != request {
		t.Errorf("got the request %q, want %q", note, request)
	}
}

func TestExpiryReadsATimeWithAnOffset(t *testing.T) {
	client, _ := standIn(t, "", expiringAt("2023-04-26 23:23:18 +0200"))

	got, err := client.Expiry(t.Context())

	want := time.Date(2023, time.April, 26, 21, 23, 18, 0, time.UTC)
	if err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v and %v, want %v", got, err, want)
	}
}

func TestExpiryIsZeroForATokenThatNeverExpires(t *testing.T) {
	client, _ := standIn(t, "", answer{status: http.StatusOK})

	got, err := client.Expiry(t.Context())
	if err != nil || !got.IsZero() {
		t.Errorf("got %v and %v, want the zero time", got, err)
	}
}

func TestExpiryFailsOnATimeThatIsNoUse(t *testing.T) {
	for _, value := range []string{
		"soon", "2027-10-27", "27/10/2027 22:00:00 UTC", "2027-10-27T22:00:00Z",
		"2027-10-27 22:00:00 CEST", "2027-10-27 22:00:00 +02:00",
	} {
		client, _ := standIn(t, "", expiringAt(value))

		got, err := client.Expiry(t.Context())

		want := "read GitHub's answer: " + expirationHeader + ` "` + value + `" is not a time`
		if err == nil || err.Error() != want || !got.IsZero() {
			t.Errorf("%q: got %v and %v, want the error %s", value, got, err, want)
		}
	}
}

func TestExpiryFailsWhenGitHubSaysNo(t *testing.T) {
	client, _ := standIn(t, "", answer{status: http.StatusUnauthorized})

	got, err := client.Expiry(t.Context())

	want := "ask GitHub when the token expires: 401 Unauthorized"
	if err == nil || err.Error() != want || !got.IsZero() {
		t.Errorf("got %v and %v, want the error %s", got, err, want)
	}
}
