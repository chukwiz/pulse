// Package middleware provides the gateway's HTTP middleware: request IDs, logging, panic recovery, rate limiting and API-key auth.
package middleware

import (
	"encoding/json"
	"net/http"
	"time"
)

type rateLimiter interface {
	Allow(key string) (bool, time.Duration)
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorCode string

const (
	codeUnauthorized  errorCode = "unauthorized"
	codeInternalError errorCode = "internal_error"
	codeRateLimited   errorCode = "rate_limited"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already sent so the error cannot change the response.
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string, code errorCode) {
	writeJSON(w, status, errorResponse{
		Error: errorBody{
			Message: message,
			Code:    string(code),
		},
	})
}
