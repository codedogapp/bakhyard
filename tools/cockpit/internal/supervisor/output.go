package supervisor

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

// output writes the lines of every service to one writer, each marked with its name, and never
// mixes two lines.
type output struct {
	mu    sync.Mutex
	out   io.Writer
	width int
	err   error // the first failed write; nothing is written after it
}

func newOutput(out io.Writer, services []manifest.Service) *output {
	width := len(cockpit)
	for _, svc := range services {
		width = max(width, len(svc.Name))
	}

	return &output{out: out, width: width}
}

// writeLine writes text as one line, marked with name.
func (o *output) writeLine(name, text string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.err == nil {
		_, o.err = fmt.Fprintf(o.out, "%-*s | %s\n", o.width, name, text)
	}
}

// failure returns the first failed write, if any.
func (o *output) failure() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.err != nil {
		return fmt.Errorf("write the output: %w", o.err)
	}

	return nil
}

// lineWriter takes what one service writes and passes it on to the output line by line.
type lineWriter struct {
	out     *output
	name    string
	pending []byte // the start of a line whose newline hasn't come yet
}

// Write never fails: an error would stop the copying of the service's output, and the service
// would block once its pipe is full. The output reports a failed write when Run ends.
func (w *lineWriter) Write(data []byte) (int, error) {
	w.pending = append(w.pending, data...)

	for {
		line, rest, found := bytes.Cut(w.pending, []byte{'\n'})
		if !found {
			return len(data), nil
		}

		w.out.writeLine(w.name, string(line))
		w.pending = rest
	}
}

// flush writes the last line, if it had no newline.
func (w *lineWriter) flush() {
	if len(w.pending) > 0 {
		w.out.writeLine(w.name, string(w.pending))
		w.pending = nil
	}
}
