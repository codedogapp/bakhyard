package service

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"log/slog"
	"slices"
	"testing"
	"time"
)

// logLine holds the keys of a JSON log line that the tests check.
type logLine struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
	Err   string    `json:"err"`
}

func TestSetLogsWritesJSONLines(t *testing.T) {
	want := []logLine{
		{Level: "ERROR", Msg: "poll Jira", Err: "connection refused"},
		// The log package's lines, such as net/http's, go through slog's default too.
		{Level: "INFO", Msg: "http: TLS handshake error"},
	}

	for _, format := range []string{"", "json"} {
		t.Run("format "+format, func(t *testing.T) {
			keepDefaultLogger(t)

			var out bytes.Buffer

			err := setLogs(logFormatEnv(format), &out)
			if err != nil {
				t.Fatal(err)
			}

			slog.Error("poll Jira", "err", "connection refused")
			log.Print("http: TLS handshake error")

			got := readLines(t, out.Bytes())
			if !slices.Equal(got, want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestSetLogsKeepsTheDefaultLoggerForText(t *testing.T) {
	keepDefaultLogger(t)

	before := slog.Default()

	err := setLogs(logFormatEnv("text"), io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if slog.Default() != before {
		t.Error("text replaced slog's default logger")
	}
}

func TestSetLogsRefusesOtherFormats(t *testing.T) {
	for _, format := range []string{"JSON", "logfmt", " json"} {
		t.Run(format, func(t *testing.T) {
			keepDefaultLogger(t)

			before := slog.Default()

			err := setLogs(logFormatEnv(format), io.Discard)

			want := fmt.Sprintf("BAKHYARD_LOG_FORMAT: %q is neither json nor text", format)
			if err == nil || err.Error() != want {
				t.Errorf("got %v, want %s", err, want)
			}

			if slog.Default() != before {
				t.Error("a format it refused replaced slog's default logger")
			}
		})
	}
}

// readLines returns the JSON lines in out, each with its time zeroed once it's checked.
func readLines(t *testing.T, out []byte) []logLine {
	t.Helper()

	var lines []logLine

	for text := range bytes.Lines(out) {
		var line logLine

		err := json.Unmarshal(text, &line)
		if err != nil {
			t.Fatalf("%q isn't a JSON line: %v", text, err)
		}

		if line.Time.IsZero() {
			t.Errorf("%q has no time", text)
		}

		line.Time = time.Time{}
		lines = append(lines, line)
	}

	return lines
}

// logFormatEnv returns a getenv in which BAKHYARD_LOG_FORMAT is format, and nothing else is set.
func logFormatEnv(format string) func(string) string {
	return func(key string) string {
		if key == "BAKHYARD_LOG_FORMAT" {
			return format
		}

		return ""
	}
}

// keepDefaultLogger restores slog's default logger once t ends, with the log package's output
// and flags, which slog.SetDefault changes too.
func keepDefaultLogger(t *testing.T) {
	t.Helper()

	logger, output, flags := slog.Default(), log.Writer(), log.Flags()

	t.Cleanup(func() {
		slog.SetDefault(logger)
		log.SetOutput(output)
		log.SetFlags(flags)
	})
}
