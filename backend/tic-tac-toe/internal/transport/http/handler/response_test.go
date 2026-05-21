package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	webresponse "tic-tac-toe/internal/transport/http/response"
)

func TestWriteUnauthorized(t *testing.T) {
	rec := httptest.NewRecorder()
	webresponse.WriteUnauthorized(rec)
	assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
}
