package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

const internalErrorMsg = "something went wrong"

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

			writeError(w, http.StatusInternalServerError, internalErrorMsg, codeInternalError)
		}()

		next.ServeHTTP(rw, r)
	})
}
