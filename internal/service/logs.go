package service

import (
	"fmt"
	"io"
	"log/slog"
)

// logFormat is the format of a service's logs, which BAKHYARD_LOG_FORMAT names (docs/setup.md).
type logFormat string

const (
	// logJSON, the default, writes JSON lines, which the laptop's VictoriaLogs reads into fields.
	logJSON logFormat = "json"
	// logText keeps slog's default logger, whose text lines process-compose's TUI shows as they are.
	logText logFormat = "text"
)

// setLogs sets slog's default logger, through which every service logs, to the format that
// BAKHYARD_LOG_FORMAT names: JSON lines on w when it's json or unset, or slog's default text
// lines on stderr, as they were, when it's text.
func setLogs(getenv func(string) string, w io.Writer) error {
	switch format := logFormat(getenv("BAKHYARD_LOG_FORMAT")); format {
	case "", logJSON:
		slog.SetDefault(slog.New(slog.NewJSONHandler(w, nil)))
	case logText:
		// slog's default logger writes text already.
	default:
		return fmt.Errorf("BAKHYARD_LOG_FORMAT: %q is neither json nor text", format)
	}

	return nil
}
