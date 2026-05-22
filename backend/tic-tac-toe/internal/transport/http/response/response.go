package response

import (
	"encoding/json"
	"net/http"

	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
)

const (
	jsonContentType    = "application/json"
	unauthorizedHeader = `Session realm="tic-tac-toe"`
)

func WriteError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, dto.ErrorResponse{Message: message})
}

func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message)
}

func WriteUnsupportedMediaType(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnsupportedMediaType, message)
}

func WriteConflict(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, message)
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message)
}

func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message)
}

func WriteMethodNotAllowed(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusMethodNotAllowed, message)
}

func WriteUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", unauthorizedHeader)
	WriteError(w, http.StatusUnauthorized, messages.Unauthorized)
}

func WriteForbidden(w http.ResponseWriter) {
	WriteError(w, http.StatusForbidden, messages.Forbidden)
}

func WriteTooManyRequests(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusTooManyRequests, message)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	writeJSONResponse(w, status, payload)
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
