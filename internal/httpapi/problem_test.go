package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := WriteJSON(recorder, http.StatusCreated, map[string]string{"id": "usr-1"})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	var body map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil || body["id"] != "usr-1" {
		t.Fatalf("body = %q, err = %v", recorder.Body.String(), err)
	}
}

func TestProblemFor_HidesInternalDetails(t *testing.T) {
	p := ProblemFor(errors.New("password=secret database exploded"), "req-1")
	if p.Status != 500 || p.Code != "INTERNAL_ERROR" || p.RequestID != "req-1" {
		t.Fatalf("problem = %#v", p)
	}
	if p.Detail == "password=secret database exploded" {
		t.Fatal("internal detail leaked")
	}
}

func TestProblemFor_MapsValidation(t *testing.T) {
	p := ProblemFor(ErrValidation, "req-2")
	if p.Status != 400 || p.Code != "VALIDATION_FAILED" {
		t.Fatalf("problem = %#v", p)
	}
}
