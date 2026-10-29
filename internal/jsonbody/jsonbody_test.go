package jsonbody_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
)

const refusal = "the body isn't a JSON object with just a title and a count"

type payload struct {
	Title string `json:"title"`
	Count int    `json:"count"`
}

// read hands body to jsonbody.Read and returns the response, what Read returned and what it
// decoded.
func read(t *testing.T, body string) (*httptest.ResponseRecorder, bool, payload) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/",
		strings.NewReader(body))
	rec := httptest.NewRecorder()

	var decoded payload

	ok := jsonbody.Read(rec, req, &decoded, refusal)

	return rec, ok, decoded
}

func TestReadDecodesTheBody(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       payload
	}{
		{"every member", `{"title":"Buy milk","count":2}`, payload{Title: "Buy milk", Count: 2}},
		{"a member missing", `{"count":2}`, payload{Count: 2}},
		{"no member", ` {} `, payload{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec, ok, got := read(t, test.body)
			if !ok || got != test.want {
				t.Errorf("got %t %+v, want true and %+v", ok, got, test.want)
			}

			if rec.Body.Len() != 0 || len(rec.Header()) != 0 {
				t.Errorf("got %d %q %v, want the response left alone",
					rec.Code, rec.Body, rec.Header())
			}
		})
	}
}

func TestReadRefusesWhatItCantRead(t *testing.T) {
	for _, test := range []struct{ name, body string }{
		{"an unknown member", `{"title":"Buy milk","x":1}`},
		{"a member's name in another case", `{"Title":"Buy milk"}`},
		{"not JSON", `nope`},
		{"a wrong type", `{"title":1}`},
		{"an array", `[]`},
		{"an empty body", ``},
		{"a cut-off object", `{"title":"Buy milk"`},
		{"two objects", `{"title":"a"}{"title":"b"}`},
		{"invalid UTF-8", "{\"title\":\"\xff\"}"},
		{"a repeated member", `{"title":"a","title":"b"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec, ok, _ := read(t, test.body)
			if ok || rec.Code != http.StatusBadRequest || rec.Body.String() != refusal+"\n" ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("got %t %d %q (%s), want false and a plain 400 with the refusal",
					ok, rec.Code, rec.Body, rec.Header().Get("Content-Type"))
			}
		})
	}
}

func TestReadTakesUpTo64KiB(t *testing.T) {
	const body = `{"title":"Buy milk"}`

	padded := strings.Repeat(" ", 64<<10-len(body)) + body
	if rec, ok, got := read(t, padded); !ok || got.Title != "Buy milk" {
		t.Errorf("64 KiB: got %t %d %q, want the body read", ok, rec.Code, rec.Body)
	}

	for _, large := range []string{" " + padded, `{"title":"` + strings.Repeat("a", 64<<10) + `"}`} {
		rec, ok, _ := read(t, large)
		if ok || rec.Code != http.StatusRequestEntityTooLarge ||
			rec.Body.String() != "Request Entity Too Large\n" ||
			rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("%d bytes: got %t %d %q (%s), want false and a plain 413",
				len(large), ok, rec.Code, rec.Body, rec.Header().Get("Content-Type"))
		}
	}
}
