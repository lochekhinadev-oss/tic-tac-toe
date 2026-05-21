package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

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

var errUnsupportedMediaType = errors.New("unsupported media type")

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
