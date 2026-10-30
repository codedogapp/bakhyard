package eventstream_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/internal/eventstream"
)

// errGone is the error of a write to a page that has gone.
var errGone = errors.New("the page has gone")

// gone is a response writer whose writes fail, as once the page has gone.
type gone struct{ http.ResponseWriter }

func (gone) Write([]byte) (int, error) { return 0, errGone }

// unflushable is a response writer that can't flush.
type unflushable struct{ http.ResponseWriter }

func TestStreamSendsThePageItsEvents(t *testing.T) {
	rec := httptest.NewRecorder()

	stream, err := eventstream.Open(rec)
	if err != nil {
		t.Fatal(err)
	}

	if err := stream.Send("message", map[string]string{"text": "Two\nlines"}); err != nil {
		t.Fatal(err)
	}

	if err := stream.Beat(); err != nil {
		t.Fatal(err)
	}

	want := ": open\n\nevent: message\ndata: {\"text\":\"Two\\nlines\"}\n\n: still here\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("sent %q, want %q", got, want)
	}

	headers := rec.Result().Header
	if headers.Get("Content-Type") != "text/event-stream" ||
		headers.Get("Cache-Control") != "no-store" {
		t.Errorf("headers %v, want an event stream that isn't stored", headers)
	}

	if !rec.Flushed {
		t.Error("the stream wasn't flushed")
	}
}

func TestStreamSendsNoEventThatJSONCantEncode(t *testing.T) {
	rec := httptest.NewRecorder()

	stream, err := eventstream.Open(rec)
	if err != nil {
		t.Fatal(err)
	}

	if err := stream.Send("message", make(chan int)); err == nil {
		t.Error("Send of a channel succeeded")
	}

	if got := rec.Body.String(); got != ": open\n\n" {
		t.Errorf("sent %q, want the open comment alone", got)
	}
}

func TestStreamFailsWhenItCantReachThePage(t *testing.T) {
	for _, test := range []struct {
		w    http.ResponseWriter
		want error
	}{
		{gone{httptest.NewRecorder()}, errGone},
		{unflushable{httptest.NewRecorder()}, http.ErrNotSupported},
	} {
		if _, err := eventstream.Open(test.w); !errors.Is(err, test.want) {
			t.Errorf("Open = %v, want %v", err, test.want)
		}
	}
}
