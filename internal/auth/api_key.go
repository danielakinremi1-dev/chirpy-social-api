package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if len(auth) == 0 {
		return "", fmt.Errorf("No Authorization parameters provided")
	}

	auth = strings.TrimPrefix(auth, "ApiKey ")

	return auth, nil
}
