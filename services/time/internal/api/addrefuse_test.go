package api_test

import (
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestAddingAWorklogRefusesABadBody(t *testing.T) {
	const (
		refusal = "the body isn't a JSON object with just a ticket, a date and minutes\n"
		ticket  = "invalid ticket: not a Jira key such as ABC-123\n"
		date    = "invalid date: not a date such as 2026-10-27\n"
		day     = "invalid date: not one of the last 7 days\n"
		minutes = "invalid minutes: not from 1 to 1440\n"
	)

	for _, test := range []struct {
		name, body string
		code       int
		want       string
	}{
		{"not JSON", "not json", http.StatusBadRequest, refusal},
		{"an array", "[]", http.StatusBadRequest, refusal},
		{
			"an unknown member", `{"ticket":"WEB-9","date":"2000-01-01","minutes":60,"hours":1}`,
			http.StatusBadRequest, refusal,
		},
		{
			"a member in capitals", `{"Ticket":"WEB-9","date":"2000-01-01","minutes":60}`,
			http.StatusBadRequest, refusal,
		},
		{
			"a fraction of a minute", `{"ticket":"WEB-9","date":"2000-01-01","minutes":1.5}`,
			http.StatusBadRequest, refusal,
		},
		{
			"minutes as a string", `{"ticket":"WEB-9","date":"2000-01-01","minutes":"60"}`,
			http.StatusBadRequest, refusal,
		},
		{"an empty object", "{}", http.StatusBadRequest, ticket},
		{"null", "null", http.StatusBadRequest, ticket},
		{
			"a key in lower case", `{"ticket":"web-9","date":"2000-01-01","minutes":60}`,
			http.StatusBadRequest, ticket,
		},
		{
			"a date without its zeros", `{"ticket":"WEB-9","date":"2000-1-1","minutes":60}`,
			http.StatusBadRequest, date,
		},
		{
			"a date that doesn't exist", `{"ticket":"WEB-9","date":"2000-02-30","minutes":60}`,
			http.StatusBadRequest, date,
		},
		{
			"a time", `{"ticket":"WEB-9","date":"2000-01-01T00:00:00Z","minutes":60}`,
			http.StatusBadRequest, date,
		},
		{"everything bad", `{"ticket":"x","date":"bad","minutes":0}`, http.StatusBadRequest, ticket},
		{
			"the date and the minutes bad", `{"ticket":"WEB-9","date":"bad","minutes":0}`,
			http.StatusBadRequest, date,
		},
		{
			"the day and the minutes bad", `{"ticket":"WEB-9","date":"1999-12-01","minutes":0}`,
			http.StatusBadRequest, day,
		},
		{
			"a body over 64 KiB",
			`{"ticket":"WEB-9","date":"2000-01-01","minutes":60` + strings.Repeat(" ", 64<<10) + "}",
			http.StatusRequestEntityTooLarge, "Request Entity Too Large\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				jira := &standIn{}

				rec := newTimeAPI(jira.backends(time.UTC), time.UTC).post(t.Context(), test.body)

				if rec.Code != test.code || rec.Body.String() != test.want {
					t.Errorf("got %d %q, want %d %q", rec.Code, rec.Body, test.code, test.want)
				}

				if keys, _, reads := jira.calls(); len(keys) != 0 || reads != 0 {
					t.Errorf("got %d adds and %d reads, want none", len(keys), reads)
				}
			})
		})
	}
}
