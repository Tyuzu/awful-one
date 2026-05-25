package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ParseJSON safely decodes JSON from an http.Request body into a target struct.
//
// Example:
//
//	var req MyStruct
//	if err := utils.ParseJSON(r, &req); err != nil {
//	    utils.RespondWithError(w, http.StatusBadRequest, err.Error())
//	    return
//	}
func ParseJSON(r *http.Request, target interface{}) error {
	if r == nil || r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()

	// limit request size to 1MB to prevent abuse
	limited := io.LimitReader(r.Body, 1<<20)

	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid JSON payload: " + err.Error())
	}

	// disallow trailing data after valid JSON object
	if decoder.More() {
		return errors.New("unexpected extra data after JSON object")
	}

	return nil
}
