package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"

	_ "tic-tac-toe/internal/logging"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

var (
	errInvalidRequestBody   = errors.New(messages.InvalidRequestBody)
	errInvalidUUID          = errors.New(messages.InvalidUUID)
	errUnsupportedMediaType = errors.New("unsupported media type")
	uuidPattern             = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	handlerLogPrefix        = "[transport/http/handler]"
)

func logHandler(format string, args ...any) {
	log.Printf(handlerLogPrefix+" "+format, args...)
}

func validateUUID(uuid string) error {
	if uuid == "" || !uuidPattern.MatchString(uuid) {
		return errInvalidUUID
	}

	return nil
}

func decodeJSONBody[T any](r *http.Request) (T, error) {
	var request T
	if err := decodeBody(r, &request); err != nil {
		return request, err
	}
	return request, nil
}

func decodeOptionalJSONBody[T any](r *http.Request, fallback T) (T, error) {
	request, err := decodeJSONBody[T](r)
	if errors.Is(err, io.EOF) {
		return fallback, nil
	}
	return request, err
}

func decodeBody(r *http.Request, request any) error {
	defer r.Body.Close()

	if !hasJSONContentType(r) {
		return errUnsupportedMediaType
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(request); err != nil {
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return errInvalidRequestBody
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errInvalidRequestBody
	}

	return nil
}

func hasJSONContentType(r *http.Request) bool {
	if r.ContentLength == 0 && len(r.TransferEncoding) == 0 {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

func writeDecodeError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, errUnsupportedMediaType) {
		webresponse.WriteUnsupportedMediaType(w, messages.UnsupportedMediaType)
		return true
	}
	return false
}
