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

// reader is jsonbody.Read or jsonbody.ReadLenient.
type reader func(http.ResponseWriter, *http.Request, any, string) bool

// readers are the readers that the tests run each case through, by name.
var readers = map[string]reader{"Read": jsonbody.Read, "ReadLenient": jsonbody.ReadLenient}

// read hands body to readBody and returns the response, what readBody returned and what it
// decoded.
func read(t *testing.T, readBody reader, body string) (*httptest.ResponseRecorder, bool, payload) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/",
		strings.NewReader(body))
	rec := httptest.NewRecorder()

	var decoded payload

	ok := readBody(rec, req, &decoded, refusal)

	return rec, ok, decoded
}

func TestReadersDecodeTheBody(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       payload
	}{
		{"every member", `{"title":"Buy milk","count":2}`, payload{Title: "Buy milk", Count: 2}},
		{"a member missing", `{"count":2}`, payload{Count: 2}},
		{"no member", ` {} `, payload{}},
	} {
		for name, readBody := range readers {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				rec, ok, got := read(t, readBody, test.body)
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
}

func TestReadersRefuseWhatTheyCantRead(t *testing.T) {
	for _, test := range []struct{ name, body string }{
		{"not JSON", `nope`},
		{"a wrong type", `{"title":1}`},
		{"an array", `[]`},
		{"an empty body", ``},
		{"a cut-off object", `{"title":"Buy milk"`},
		{"two objects", `{"title":"a"}{"title":"b"}`},
		{"invalid UTF-8", "{\"title\":\"\xff\"}"},
		{"a repeated member", `{"title":"a","title":"b"}`},
		{"a repeated unknown member", `{"x":1,"x":2}`},
	} {
		for name, readBody := range readers {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				assertRefused(t, readBody, test.body)
			})
		}
	}
}

func TestOnlyReadRefusesUnknownMembers(t *testing.T) {
	for _, test := range []struct {
		name, body string
		lenient    payload
	}{
		{"an unknown member", `{"title":"Buy milk","x":1}`, payload{Title: "Buy milk"}},
		{"a member's name in another case", `{"Title":"Buy milk","count":2}`, payload{Count: 2}},
		{"a nested unknown member", `{"x":{"y":[1,{"z":null}]},"count":2}`, payload{Count: 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertRefused(t, jsonbody.Read, test.body)

			if rec, ok, got := read(t, jsonbody.ReadLenient, test.body); !ok ||
				got != test.lenient || rec.Body.Len() != 0 {
				t.Errorf("ReadLenient: got %t %+v %d %q, want true and %+v",
					ok, got, rec.Code, rec.Body, test.lenient)
			}
		})
	}
}

func TestReadersTakeUpTo64KiB(t *testing.T) {
	const body = `{"title":"Buy milk"}`

	padded := strings.Repeat(" ", 64<<10-len(body)) + body
	large := []string{" " + padded, `{"title":"` + strings.Repeat("a", 64<<10) + `"}`}

	for name, readBody := range readers {
		if rec, ok, got := read(t, readBody, padded); !ok || got.Title != "Buy milk" {
			t.Errorf("%s, 64 KiB: got %t %d %q, want the body read", name, ok, rec.Code, rec.Body)
		}

		for _, tooLarge := range large {
			rec, ok, _ := read(t, readBody, tooLarge)
			if ok || rec.Code != http.StatusRequestEntityTooLarge ||
				rec.Body.String() != "Request Entity Too Large\n" ||
				rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Errorf("%s, %d bytes: got %t %d %q (%s), want false and a plain 413",
					name, len(tooLarge), ok, rec.Code, rec.Body, rec.Header().Get("Content-Type"))
			}
		}
	}
}

// assertRefused checks that readBody refuses body with a plain 400 that gives the refusal.
func assertRefused(t *testing.T, readBody reader, body string) {
	t.Helper()

	rec, ok, _ := read(t, readBody, body)
	if ok || rec.Code != http.StatusBadRequest || rec.Body.String() != refusal+"\n" ||
		rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("got %t %d %q (%s), want false and a plain 400 with the refusal",
			ok, rec.Code, rec.Body, rec.Header().Get("Content-Type"))
	}
}
