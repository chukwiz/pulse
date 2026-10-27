package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

const internalErrorBody = `{"error": {"code":"internal_error","message":"something went wrong"}}` + "\n"

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := newResponseWriterWrapper(w)
		defer func() {
			p := recover()
			if p == nil {
				return
			}

			if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(p)
			}

			LoggerFrom(r.Context()).Error("panic recovered",
				slog.Any("panic", p),
				slog.String("stack_trace", string(debug.Stack())),
			)

			if rw.wroteHeader {
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(internalErrorBody))
		}()

		next.ServeHTTP(rw, r)
	})
}
