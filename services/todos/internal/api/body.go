package api

import (
	"encoding/json/v2"
	"errors"
	"net/http"
)

// maxBody is the largest body the todos' API reads: 64 KiB, many times a 500-character title's.
const maxBody = 64 << 10

// readBody decodes r's body, a JSON object without members that v lacks, into v. When it can't,
// the client gets a 413 for a body over maxBody or a 400 with refusal otherwise, and readBody
// returns false.
func readBody(w http.ResponseWriter, r *http.Request, v any, refusal string) bool {
	err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxBody), v,
		json.RejectUnknownMembers(true))
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge)

		return false
	}

	if err != nil {
		http.Error(w, refusal, http.StatusBadRequest)

		return false
	}

	return true
}
