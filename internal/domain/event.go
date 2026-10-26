package domain

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

const (
	maxTypeLen   = 128        // bytes; the type pattern is ASCII-only, so bytes == characters
	MaxDataBytes = 256 * 1024 // 256 KB
)

// ErrInvalidEvent is the category for every validation failure.
// Callers check errors.Is(err, ErrInvalidEvent) to map it to a 400; the specific errors below wrap and say what was wrong
var ErrInvalidEvent = errors.New("invalid event")

var (
	ErrTypeTooLong   = fmt.Errorf("%w: type must be at most %d bytes", ErrInvalidEvent, maxTypeLen)
	ErrInvalidType   = fmt.Errorf("%w: type must be dot-separated segments of a-z, 0-9 and _", ErrInvalidEvent)
	ErrDataTooLarge  = fmt.Errorf("%w: data must be at most %d KB", ErrInvalidEvent, MaxDataBytes/1024)
	ErrDataNotJSON   = fmt.Errorf("%w: data must be valid JSON", ErrInvalidEvent)
	ErrDataNotObject = fmt.Errorf("%w: data must be a JSON object", ErrInvalidEvent)
)

var (
	// ErrNotFound means no event exists with the requested ID.
	ErrNotFound = errors.New("event not found")
	// ErrInvalidCursor means a list cursor does not refer to a known event.
	ErrInvalidCursor = errors.New("invalid cursor")
)

var validEventType = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)*$`)

// Event represents a single event accepted from a sender.
type Event struct {
	ID             string
	Type           string
	Data           json.RawMessage
	APIKeyID       string
	IdempotencyKey string
	CreatedAt      time.Time
}

// NewEvent creates a new event using user provided data
func NewEvent(eventType, idempotencyKey, apiKey string, data json.RawMessage) (Event, error) {
	randID := generatePrefixedID("evt", 16)
	event := Event{
		ID:             randID,
		Type:           eventType,
		Data:           data,
		IdempotencyKey: idempotencyKey,
		APIKeyID:       apiKey,
		CreatedAt:      time.Now().UTC(),
	}

	return event, event.Validate()
}

// Validate checks the fields sent by the sender (Type and Data).
// Every error it returns matches ErrInvalidEvent
func (e Event) Validate() error {
	if len(e.Type) > maxTypeLen {
		return ErrTypeTooLong
	}
	if !validEventType.MatchString(e.Type) {
		return ErrInvalidType
	}

	if len(e.Data) > MaxDataBytes {
		return ErrDataTooLarge
	}

	// Check syntax without decoding.Data is passed through untouched, so
	// building a map just to inspect its top-level type would be wasted work.
	if !json.Valid(e.Data) {
		return ErrDataNotJSON
	}

	// Valid JSON whose first non-whitespace byte is '{' is an object.
	if trimmed := bytes.TrimLeft(e.Data, " \t\r\n"); trimmed[0] != '{' {
		return ErrDataNotObject
	}
	return nil
}

// generatePrefixedID generates a unique id for every event
func generatePrefixedID(prefix string, byteLength int) string {
	b := make([]byte, byteLength)
	_, _ = rand.Read(b)

	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
