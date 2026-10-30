// Package jsonbody reads the JSON body of a request to a service's API, up to 64 KiB (decision
// 190).
package jsonbody

import (
	"encoding/json/v2"
	"errors"
	"net/http"
)

// maxBody is the largest body that Read and ReadLenient read: many times what any of the bodies
// needs.
const maxBody = 64 << 10

// Read decodes r's body, a JSON object without members that v lacks, into v. When it can't, the
// client gets a 413 for a body over 64 KiB or a 400 with refusal otherwise, and Read returns
// false.
func Read(w http.ResponseWriter, r *http.Request, v any, refusal string) bool {
	return answer(w, json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxBody), v,
		json.RejectUnknownMembers(true)), refusal)
}

// ReadLenient is Read for a body whose sender adds members over time, such as a webhook's: it
// skips the members that v lacks.
func ReadLenient(w http.ResponseWriter, r *http.Request, v any, refusal string) bool {
	return answer(w, json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxBody), v), refusal)
}

// answer tells the client why its body couldn't be read, when err says it couldn't, and returns
// whether it was read.
func answer(w http.ResponseWriter, err error, refusal string) bool {
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
