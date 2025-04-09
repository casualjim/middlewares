package middlewares

import (
	"log/slog"
	"net/http"
	"runtime"
	"sync/atomic"

	"github.com/casualjim/middlewares/slogx"
	"github.com/felixge/httpsnoop"
)

type PanicRenderer func(http.ResponseWriter, string, int, ...http.Header)

func Recover(lg *slog.Logger) func(http.Handler) http.Handler {
	return RecoverRendered(lg, nil)
}

func RecoverRendered(lg *slog.Logger, renderPanic PanicRenderer) func(http.Handler) http.Handler {
	if renderPanic == nil {
		renderPanic = JSONError
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusWritten := int32(-1)
			w = httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(headerFunc httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(code int) {
						if atomic.CompareAndSwapInt32(&statusWritten, -1, int32(code)) {
							headerFunc(code)
						}
					}
				},
			})

			defer func() {
				if rvr := recover(); rvr != nil {
					stack := make([]byte, 8*1024)
					stack = stack[:runtime.Stack(stack, false)]

					if err, ok := rvr.(error); ok {
						slog.Warn("", slogx.Error(err), slogx.ByteString("stack", stack))
						renderPanic(w, string(stack), http.StatusInternalServerError)
					} else {
						slog.Info("", slog.Any("error", rvr), slogx.ByteString("stack", stack))
						renderPanic(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
