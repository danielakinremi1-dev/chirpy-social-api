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

	auth = strings.TrimPrefix(auth, "Bearer ")

	return auth, nil
}
