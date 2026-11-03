package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if len(auth) == 0 {
		return "", fmt.Errorf("No Authorization parameters provided")
	}

	fields := strings.Fields(auth)
	if len(fields) != 2 || fields[0] != "Bearer" {
		return "", fmt.Errorf("Malformed Authorization header")
	}

	return fields[1], nil
}
