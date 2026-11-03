package api

import (
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{email: "user@example.com", wantErr: false},
		{email: "first.last+tag@sub.example.com", wantErr: false},
		{email: "", wantErr: true},
		{email: "not-an-email", wantErr: true},
		{email: "user@", wantErr: true},
		{email: "Name <user@example.com>", wantErr: true},
		{email: " user@example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "Empty", password: "", wantErr: true},
		{name: "Too short", password: "1234567", wantErr: true},
		{name: "Minimum length", password: "12345678", wantErr: false},
		{name: "Multibyte characters count once", password: "пароль12", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestFilterBadWords(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "Lowercase", text: "what a kerfuffle", want: "what a ****"},
		{name: "Mixed case", text: "KerFUFFle and SHARBERT", want: "**** and ****"},
		{name: "Punctuation", text: "fornax! really", want: "****! really"},
		{name: "Substring is kept", text: "kerfuffles and sharberts", want: "kerfuffles and sharberts"},
		{name: "Clean text", text: "hello world", want: "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterBadWords(tt.text); got != tt.want {
				t.Errorf("filterBadWords(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "Unique violation", err: &pq.Error{Code: "23505"}, want: true},
		{name: "Wrapped unique violation", err: fmt.Errorf("insert: %w", &pq.Error{Code: "23505"}), want: true},
		{name: "Other Postgres error", err: &pq.Error{Code: "23503"}, want: false},
		{name: "Non-Postgres error", err: fmt.Errorf("boom"), want: false},
		{name: "Nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueViolation(tt.err); got != tt.want {
				t.Errorf("isUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
