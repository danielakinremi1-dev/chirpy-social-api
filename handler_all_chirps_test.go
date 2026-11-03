package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChirpCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 11, 3, 6, 9, 37, 205220000, time.FixedZone("EST", -5*60*60))
	id := uuid.New()

	gotCreatedAt, gotID, err := decodeChirpCursor(encodeChirpCursor(createdAt, id))
	if err != nil {
		t.Fatalf("decodeChirpCursor() error = %v", err)
	}
	if !gotCreatedAt.Equal(createdAt) {
		t.Errorf("decodeChirpCursor() createdAt = %v, want %v", gotCreatedAt, createdAt)
	}
	if gotID != id {
		t.Errorf("decodeChirpCursor() id = %v, want %v", gotID, id)
	}
}

func TestDecodeChirpCursorInvalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{name: "Not base64", cursor: "!!!"},
		{name: "Missing separator", cursor: "bm9jb21tYQ"},
		{name: "Bad time", cursor: "bm90LWEtdGltZSwwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDA"},
		{name: "Bad UUID", cursor: "MjAyNi0xMS0wM1QxMTowOTozN1osbm90LWEtdXVpZA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := decodeChirpCursor(tt.cursor); err == nil {
				t.Errorf("decodeChirpCursor(%q) expected error", tt.cursor)
			}
		})
	}
}
