package auth

import "strings"

const bearerPrefix = "Bearer "

func parseBearerAuthorizationHeader(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", ErrInvalidAuthHeader
	}

	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(bearerPrefix)) {
		return "", ErrInvalidAuthHeader
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", ErrInvalidAuthHeader
	}
	return token, nil
}
