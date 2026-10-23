package event

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID             string
	Type           string
	Data           json.RawMessage
	APIKeyID       string
	IdempotencyKey string
	CreatedAt      time.Time
}

func (e *Event) Validate() {
	
}