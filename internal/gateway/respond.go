package gateway

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorCode string

const (
	codeInvalidJSON           errorCode = "invalid_json"
	codeInvalidEvent          errorCode = "invalid_event"
	codeMissingIdempotencyKey errorCode = "missing_idempotency_key"
	codePayloadTooLarge       errorCode = "payload_too_large"
	codeEventNotFound         errorCode = "not_found"
	codeInternalError         errorCode = "internal_error"
	codeInvalidQueryParameter errorCode = "invalid_query_parameter"
	codeInvalidCursor         errorCode = "invalid_cursor"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string, code errorCode) {
	writeJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    string(code),
			Message: message,
		},
	})
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.logger.Error(op+" failed", "err", err, "method", r.Method, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, "something went wrong", codeInternalError)
}
