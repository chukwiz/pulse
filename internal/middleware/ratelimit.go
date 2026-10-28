package middleware

import (
	"math"
	"net/http"
	"strconv"
)

func RateLimit(limiter rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := APIKeyIDFrom(r.Context())
			if id == "" {
				logger := LoggerFrom(r.Context())
				logger.Error("no API key ID on request; is RateLimit mounted inside Auth?")
				writeError(w, http.StatusInternalServerError, "something went wrong", codeInternalError)
				return
			}

			allowed, wait := limiter.Allow(id)
			if !allowed {
				retryAfter := strconv.Itoa(max(1, int(math.Ceil(wait.Seconds()))))
				w.Header().Set("Retry-After", retryAfter)
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded", codeRateLimited)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
