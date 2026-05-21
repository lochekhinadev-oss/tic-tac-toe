package handler

import (
	"net/http"
)

func writeAuthError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error, write func(http.ResponseWriter, string)) bool {
	if err == nil {
		return false
	}

	if responseMessage == "" {
		responseMessage = err.Error()
	}
	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	write(w, responseMessage)
	return true
}
