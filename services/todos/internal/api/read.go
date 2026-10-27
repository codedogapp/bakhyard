package api

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
)

// serveRead returns the handler of a GET that answers what read returns, as JSON. When read
// fails, the client gets a 500, and logger gets failure and the cause.
func serveRead[T any](
	read func(context.Context) (T, error),
	failure string,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := read(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), failure, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		// What read returns holds text from the UTF8 database alone, so it always encodes: an
		// error means the client has gone.
		_ = json.MarshalWrite(w, body)
	}
}
