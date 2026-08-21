package httpapi

import (
	"errors"
	"net/http"
)

var ErrValidation = errors.New("validation failed")

type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Detail    string `json:"detail"`
	RequestID string `json:"request_id"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	// TODO(level-04): set content type, write status, and encode exactly one JSON value.
	return errors.New("TODO(level-04)")
}

func ProblemFor(err error, requestID string) Problem {
	// TODO(level-04): map known errors and hide unknown internal error details.
	return Problem{}
}
