package supervisor

import (
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

func TestLineWriterPassesOnWholeLines(t *testing.T) {
	var out strings.Builder

	lines := &lineWriter{out: newOutput(&out, []manifest.Service{{Name: "home"}}), name: "home"}

	for _, chunk := range []string{"one\ntw", "", "o\n\nthree\nfo", "ur"} {
		n, err := lines.Write([]byte(chunk))
		if n != len(chunk) || err != nil {
			t.Fatalf("Write(%q) = %d, %v", chunk, n, err)
		}
	}

	lines.flush()
	lines.flush()

	want := "home    | one\nhome    | two\nhome    | \nhome    | three\nhome    | four\n"
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}
}
