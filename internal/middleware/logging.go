package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const loggerKey contextKey = "slogLogger"

func LoggerFrom(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode  int
	bytes       int
	wroteHeader bool
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriterWrapper) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	rw.wroteHeader = true
	size, err := rw.ResponseWriter.Write(b)
	rw.bytes += size
	return size, err
}

func ContextLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopedLogger := baseLogger.With(
				slog.String("request_id", RequestIDFrom(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)

			r = r.WithContext(context.WithValue(r.Context(), loggerKey, scopedLogger))

			wrapper := newResponseWriterWrapper(w)
			start := time.Now()
			next.ServeHTTP(wrapper, r)
			duration := time.Since(start)

			scopedLogger.Info("request completed",
				slog.Int("status", wrapper.statusCode),
				slog.Int("bytes", wrapper.bytes),
				slog.Float64("duration_ms", float64(duration.Microseconds())/1000),
			)
		})
	}
}
