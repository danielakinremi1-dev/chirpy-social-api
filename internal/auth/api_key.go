package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if len(auth) == 0 {
		return "", errors.New("no authorization header provided")
	}

	fields := strings.Fields(auth)
	if len(fields) != 2 || fields[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}

	return fields[1], nil
}
