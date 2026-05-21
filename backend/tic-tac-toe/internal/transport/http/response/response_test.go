package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tic-tac-toe/internal/transport/http/dto"
)

func TestWriteTooManyRequests(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteTooManyRequests(rec, "too many")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	var payload dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != "too many" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestWriteHelpers(t *testing.T) {
	cases := []struct {
		name     string
		write    func(http.ResponseWriter)
		status   int
		message  string
		header   string
		value    string
		expected string
	}{
		{name: "bad request", write: func(w http.ResponseWriter) { WriteBadRequest(w, "bad") }, status: http.StatusBadRequest, message: "bad"},
		{name: "conflict", write: func(w http.ResponseWriter) { WriteConflict(w, "conflict") }, status: http.StatusConflict, message: "conflict"},
		{name: "not found", write: func(w http.ResponseWriter) { WriteNotFound(w, "missing") }, status: http.StatusNotFound, message: "missing"},
		{name: "internal error", write: func(w http.ResponseWriter) { WriteInternalError(w, "boom") }, status: http.StatusInternalServerError, message: "boom"},
		{name: "method not allowed", write: func(w http.ResponseWriter) { WriteMethodNotAllowed(w, "nope") }, status: http.StatusMethodNotAllowed, message: "nope"},
		{name: "too many requests", write: func(w http.ResponseWriter) { WriteTooManyRequests(w, "slow down") }, status: http.StatusTooManyRequests, message: "slow down"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.write(rec)
			assertErrorResponse(t, rec, tc.status, tc.message)
		})
	}
}

func TestWriteUnauthorizedAndJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteUnauthorized(rec)
	assertErrorResponse(t, rec, http.StatusUnauthorized, "unauthorized")
	if got := rec.Header().Get("WWW-Authenticate"); got != unauthorizedHeader {
		t.Fatalf("unexpected auth header: %q", got)
	}

	rec = httptest.NewRecorder()
	WriteJSON(rec, http.StatusAccepted, map[string]string{"status": "ok"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != jsonContentType {
		t.Fatalf("unexpected content type: %q", got)
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("expected %d, got %d", wantStatus, rec.Code)
	}
	var payload dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != wantMessage {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
