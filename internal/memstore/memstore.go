package memstore

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/chukwiz/pulse/internal/domain"
)

type idemKey struct {
	apiKeyID       string
	idempotencyKey string
}

type Store struct {
	mu        sync.RWMutex
	events    map[string]domain.Event // event ID → event
	byIdemKey map[idemKey]string      // (api key, idempotency key) → event ID
	order     []string                // event IDs in insertion order, for List
}

func New() *Store {
	return &Store{
		events:    make(map[string]domain.Event),
		byIdemKey: make(map[idemKey]string),
	}
}

func (s *Store) Save(ctx context.Context, e domain.Event) (domain.Event, bool, error) {
	key := idemKey{
		apiKeyID:       e.APIKeyID,
		idempotencyKey: e.IdempotencyKey,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.byIdemKey[key]; ok {
		return clone(s.events[id]), false, nil
	}

	if _, ok := s.events[e.ID]; ok {
		return domain.Event{}, false, fmt.Errorf("memstore: %q already exsits", e.ID)
	}

	e = clone(e)
	s.events[e.ID] = e
	s.byIdemKey[key] = e.ID

	s.order = append(s.order, e.ID)

	return clone(e), true, nil
}

func (s *Store) Get(ctx context.Context, id string) (domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.events[id]
	if !ok {
		return domain.Event{}, domain.ErrNotFound
	}
	return clone(e), nil
}

func (s *Store) List(ctx context.Context, eventType, cursor string, limit int) ([]domain.Event, string, error) {

	if limit <= 0 {
		return nil, "", fmt.Errorf("memstore: limit must be positive, got %d", limit)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	start := 0

	if cursor != "" {
		i := slices.Index(s.order, cursor)
		if i == -1 {
			return nil, "", domain.ErrInvalidCursor
		}

		start = i + 1
	}

	page := make([]domain.Event, 0, limit+1)
	for _, id := range s.order[start:] {
		e := s.events[id]
		if eventType != "" && e.Type != eventType {
			continue
		}
		page = append(page, clone(e))
		if len(page) > limit {
			break
		}
	}

	next := ""
	if len(page) > limit {
		page = page[:limit]
		next = page[limit-1].ID
	}
	return page, next, nil
}

func clone(e domain.Event) domain.Event {
	e.Data = bytes.Clone(e.Data)
	return e
}
