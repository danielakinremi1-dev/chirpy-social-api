package api

import (
	"errors"
	"fmt"
	"net/mail"
	"unicode/utf8"

	"github.com/lib/pq"
)

const minPasswordLength = 8

// credentials is the email/password body shared by the user and login endpoints.
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// validate returns an error, safe to show to clients, if the credentials can't
// be used for an account.
func (c credentials) validate() error {
	if err := validateEmail(c.Email); err != nil {
		return err
	}
	return validatePassword(c.Password)
}

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	// ParseAddress also accepts display names like "Name <a@b.com>"; only allow the bare address.
	if err != nil || addr.Address != email {
		return errors.New("invalid email address")
	}
	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation"
}
