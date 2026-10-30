// Package eventstream sends a page server-sent events, which the browser's EventSource reads
// (decision 408).
package eventstream

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
)

// Stream is an event stream to a page.
type Stream struct {
	w       http.ResponseWriter
	flusher *http.ResponseController
}

// Open opens an event stream on w: it sends the headers, and a comment so that the page knows at
// once that the stream is open.
func Open(w http.ResponseWriter) (Stream, error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")

	stream := Stream{w: w, flusher: http.NewResponseController(w)}

	return stream, stream.write(": open\n\n")
}

// Send sends the page the event name, with data as JSON.
func (s Stream) Send(name string, data any) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode the event %s: %w", name, err)
	}

	return s.write("event: " + name + "\ndata: " + string(encoded) + "\n\n")
}

// Beat sends a comment, so that the proxies in between don't close the stream while it's idle.
func (s Stream) Beat() error {
	return s.write(": still here\n\n")
}

// write sends message to the page at once.
func (s Stream) write(message string) error {
	if _, err := io.WriteString(s.w, message); err != nil {
		return fmt.Errorf("write to the event stream: %w", err)
	}

	if err := s.flusher.Flush(); err != nil {
		return fmt.Errorf("flush the event stream: %w", err)
	}

	return nil
}
