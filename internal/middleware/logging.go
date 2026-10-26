package middleware

import (
	"log"
	"net/http"
	"time"
)

type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.bodySize += size
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &ResponseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		reqID := ""
		if id, ok := r.Context().Value(requestIDKey).(string); ok {
			reqID = id
		}

		log.Printf("[%s%s] %s %s %d %dB in %v",
			reqID,
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapper.statusCode,
			wrapper.bodySize,
			duration,
		)
	})
}
