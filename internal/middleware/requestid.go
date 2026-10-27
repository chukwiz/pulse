package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
)

type contextKey string

const (
	requestIDKey    contextKey = "requestID"
	requestIDPrefix string     = "req"
)

var strictRequestIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{8,64}$`)

func RequestIDFrom(ctx context.Context) string {
	reqID, _ := ctx.Value(requestIDKey).(string)
	return reqID
}

func generateRequestID() string {
	b := make([]byte, 16)

	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%x", requestIDPrefix, b)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestIDHeader := r.Header.Get("X-Request-ID")

		requestID := validateOrCreateRequestID(requestIDHeader)

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func validateOrCreateRequestID(id string) string {
	if !strictRequestIDRegex.MatchString(id) {
		return generateRequestID()
	}
	return id
}
