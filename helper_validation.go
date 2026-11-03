package main

import (
	"errors"
	"fmt"
	"net/mail"
	"unicode/utf8"

	"github.com/lib/pq"
)

const minPasswordLength = 8

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	// ParseAddress also accepts display names like "Name <a@b.com>"; only allow the bare address.
	if err != nil || addr.Address != email {
		return fmt.Errorf("invalid email address: %q", email)
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
