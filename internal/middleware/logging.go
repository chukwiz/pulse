package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const loggerKey contextKey = "slogLogger"

func LoggerFrom(ctx context.Context) *slog.Logger {
	var logger *slog.Logger
	if val, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		logger = val
	} else {
		logger = slog.Default()
	}
	return logger
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func (rw *responseWriterWrapper) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.bodySize += size
	return size, err
}

func ContextLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			reqID := ""
			if id, ok := r.Context().Value(requestIDKey).(string); ok {
				reqID = id
			}

			scopedLogger := baseLogger.With(
				slog.String("request_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)

			ctx := context.WithValue(r.Context(), loggerKey, scopedLogger)

			r = r.WithContext(ctx)

			start := time.Now()
			next.ServeHTTP(wrapper, r)

			duration := time.Since(start)
			scopedLogger.Info("request completed",
				slog.Int("status", wrapper.statusCode),
				slog.Int("bytes", wrapper.bodySize), slog.Float64("duration_ms", float64(duration.Microseconds())))

		})
	}
}
