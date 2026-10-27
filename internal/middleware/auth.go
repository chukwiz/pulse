package middleware

import (
	"context"
	"crypto/sha256"
	"net/http"

	"github.com/chukwiz/pulse/internal/config"
)

const (
	apiKeyIDKey          contextKey = "apiKeyID"
	unauthorizedErrorMsg string     = "missing or invalid API key"
)

func APIKeyIDFrom(ctx context.Context) string {
	apiKeyID, _ := ctx.Value(apiKeyIDKey).(string)
	return apiKeyID
}

func Auth(keys []config.APIKey) func(http.Handler) http.Handler {
	hashTable := make(map[[32]byte]string, len(keys))
	for _, k := range keys {
		hash := sha256.Sum256([]byte(k.Key))
		hashTable[hash] = k.ID
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")

			if apiKey == "" {
				writeError(w, http.StatusUnauthorized, unauthorizedErrorMsg, codeUnauthorized)
				return
			}

			keyHash := sha256.Sum256([]byte(apiKey))

			id, ok := hashTable[keyHash]
			if !ok {
				writeError(w, http.StatusUnauthorized, unauthorizedErrorMsg, codeUnauthorized)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), apiKeyIDKey, id))

			next.ServeHTTP(w, r)
		})
	}
}
