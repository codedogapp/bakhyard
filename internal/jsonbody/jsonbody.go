// Package jsonbody reads the JSON body of a request to a service's API, up to 64 KiB (decision
// 190).
package jsonbody

import (
	"encoding/json/v2"
	"errors"
	"net/http"
)

// maxBody is the largest body that Read reads: many times what any of the bodies needs.
const maxBody = 64 << 10

// Read decodes r's body, a JSON object without members that v lacks, into v. When it can't, the
// client gets a 413 for a body over 64 KiB or a 400 with refusal otherwise, and Read returns
// false.
func Read(w http.ResponseWriter, r *http.Request, v any, refusal string) bool {
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
