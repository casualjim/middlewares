package middlewares

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"slices"
	"time"

	"github.com/casualjim/middlewares/slogx"
	"github.com/felixge/httpsnoop"
)

type (
	contextRequestStartT struct{}
	contextHandlerStartT struct{}
)

var (
	contextRequestStart contextRequestStartT
	contextHandlerStart contextHandlerStartT
)

func AlwaysDumpRequestBody() func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpRequestBody = func(*http.Request) bool { return true }
	}
}

func AlwaysDumpResponseBody() func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpResponseBody = func(*http.Request, *http.Response) bool { return true }
	}
}

func NeverDumpRequestBody() func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpRequestBody = func(*http.Request) bool { return false }
	}
}

func NeverDumpResponseBody() func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpResponseBody = func(*http.Request, *http.Response) bool { return false }
	}
}

func FilteredDumpRequestBody(filter func(*http.Request) bool) func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpRequestBody = filter
	}
}

func FilteredDumpResponseBody(filter func(*http.Request, *http.Response) bool) func(*loggingTransport) {
	return func(t *loggingTransport) {
		t.dumpResponseBody = filter
	}
}

// LoggingTransport decorates an existing transport with logging of request and responses
func LoggingTransport(toWrap http.RoundTripper, opts ...func(*loggingTransport)) http.RoundTripper {
	tr := &loggingTransport{
		w:               toWrap,
		dumpRequestBody: func(*http.Request) bool { return false },
		dumpResponseBody: func(*http.Request, *http.Response) bool {
			return false
		},
	}
	for opt := range slices.Values(opts) {
		opt(tr)
	}
	return tr
}

func LoggingTransportDebug(toWrap http.RoundTripper, opts ...func(*loggingTransport)) http.RoundTripper {
	tr := &loggingTransport{
		w:                toWrap,
		dumpRequestBody:  func(*http.Request) bool { return true },
		dumpResponseBody: func(*http.Request, *http.Response) bool { return true },
	}

	for opt := range slices.Values(opts) {
		opt(tr)
	}

	return tr
}

type loggingTransport struct {
	w                http.RoundTripper
	dumpRequestBody  func(*http.Request) bool
	dumpResponseBody func(*http.Request, *http.Response) bool
}

func (l *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), contextRequestStart, time.Now())

	lg := slog.With("loggerName", "http.client", "method", req.Method, "uri", req.URL.String())
	req = req.WithContext(ctx)

	b, err := httputil.DumpRequest(req, l.dumpRequestBody(req))
	if err != nil {
		return nil, err
	}

	lg.Info("request " + req.Method + " " + req.URL.String())
	slog.Debug("request:\n" + string(b))

	resp, err := l.w.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	b, derr := httputil.DumpResponse(resp, l.dumpResponseBody(req, resp))
	if derr != nil {
		return resp, derr
	}

	if start, ok := ctx.Value(contextRequestStart).(time.Time); ok {
		lg.Info("response "+req.Method+" "+req.URL.String(), "status", resp.StatusCode, "elapsed", time.Since(start))
	} else {
		lg.Info("response "+req.Method+" "+req.URL.String(), "status", resp.StatusCode)
	}
	slog.Debug("response:\n" + string(b))

	return resp, err
}

// DebugDumpMiddleware that logs the request and responses.
func DebugDumpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), contextRequestStart, time.Now())
		r = r.WithContext(ctx)

		b, err := httputil.DumpRequest(r, true)
		if err != nil {
			slog.Error("dumping request for debug", slogx.Error(err))
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		reqMsg := fmt.Sprintf("request %s %s", r.Method, r.RequestURI)
		slog.Info(reqMsg, slog.String("method", r.Method), slog.String("uri", r.RequestURI), slog.Any("headers", r.Header), slogx.ByteString("body", b))

		var statusCode int
		body := bytes.NewBuffer(nil)
		nextw := httpsnoop.Wrap(rw, httpsnoop.Hooks{
			WriteHeader: func(whf httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(code int) {
					statusCode = code
					whf(code)
				}
			},
			Write: func(wf httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(b []byte) (int, error) {
					body.Write(b)
					return wf(b)
				}
			},
		})

		next.ServeHTTP(nextw, r)

		respMsg := fmt.Sprintf("response [%d] %s %s", statusCode, r.Method, r.RequestURI)
		if start, ok := ctx.Value(contextRequestStart).(time.Time); ok {
			slog.Info(respMsg, slog.Int("status", statusCode), slog.String("uri", r.RequestURI), slog.Duration("elapsed", time.Since(start)), slog.Any("headers", rw.Header()), slogx.Stringer("body", body))
		} else {
			slog.Info(respMsg, slog.Int("status", statusCode), slog.String("uri", r.RequestURI), slog.Any("headers", rw.Header()), slog.String("body", body.String()))
		}
	})
}

// // LogMiddleware that logs the request and responses.
// func LogMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
// 		lg := zlog.Ctx(r.Context()).With(slog.String("method", r.Method), slog.String("url", r.RequestURI), slog.String("remote", r.RemoteAddr))
// 		ctx := context.WithValue(r.Context(), contextHandlerStart, time.Now())

// 		lg = captureIPCFields(ctx, r, lg)
// 		r = r.WithContext(ctx)

// 		reqMsg := fmt.Sprintf("request %s %s", r.Method, r.RequestURI)
// 		lg.Debug(reqMsg)

// 		statusCode := http.StatusOK
// 		contentLength := int64(0)
// 		nextw := httpsnoop.Wrap(rw, httpsnoop.Hooks{
// 			WriteHeader: func(whf httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
// 				return func(code int) {
// 					statusCode = code
// 					whf(code)
// 				}
// 			},
// 			Write: func(wf httpsnoop.WriteFunc) httpsnoop.WriteFunc {
// 				return func(b []byte) (int, error) {
// 					n, err := wf(b)
// 					contentLength += int64(n)
// 					return n, err
// 				}
// 			},
// 		})

// 		next.ServeHTTP(nextw, r)
// 		fields := []slog.Field{slog.Int("status", statusCode)}
// 		if start, ok := ctx.Value(contextHandlerStart).(time.Time); ok {
// 			fields = append(fields, slog.Duration("elapsed", time.Since(start)), slog.Int64("contentLength", contentLength))
// 		}
// 		respMsg := fmt.Sprintf("response [%d] %s %s", statusCode, r.Method, r.RequestURI)
// 		lg.Info(respMsg, fields...)
// 	})
// }

// // LogDumpErrorsMiddleware that logs the request and responses.
// // This is a hybrid between the LogMiddleware and DebugDumpMiddleware
// // It logs the request and response for all requests, but only logs the response body for requests that return an error.
// func LogDumpErrorsMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
// 		ctx := context.WithValue(r.Context(), contextRequestStart, time.Now())
// 		lg := zlog.Ctx(ctx).With(slog.String("method", r.Method), slog.String("url", r.RequestURI), slog.String("remote", r.RemoteAddr))
// 		lg = captureIPCFields(ctx, r, lg)
// 		r = r.WithContext(ctx)

// 		reqMsg := fmt.Sprintf("request %s %s", r.Method, r.RequestURI)
// 		lg.Debug(reqMsg)

// 		var statusCode int
// 		contentLength := int64(0)
// 		var body *bytes.Buffer
// 		defer func() {
// 			if body != nil {
// 				putBuffer(body)
// 			}
// 		}()

// 		nextw := httpsnoop.Wrap(rw, httpsnoop.Hooks{
// 			WriteHeader: func(whf httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
// 				return func(code int) {
// 					statusCode = code
// 					whf(code)
// 				}
// 			},
// 			Write: func(wf httpsnoop.WriteFunc) httpsnoop.WriteFunc {
// 				return func(b []byte) (int, error) {
// 					if statusCode >= 400 && statusCode < 600 {
// 						if body == nil {
// 							body = getBuffer()
// 						}
// 						_, _ = body.Write(b)
// 					}
// 					n, err := wf(b)
// 					contentLength += int64(n)
// 					return n, err
// 				}
// 			},
// 		})

// 		next.ServeHTTP(nextw, r)

// 		fields := []slog.Field{slog.Int("status", statusCode)}
// 		if statusCode >= 400 && statusCode < 600 {
// 			if start, ok := ctx.Value(contextHandlerStart).(time.Time); ok {
// 				fields = append(fields, slog.Duration("elapsed", time.Since(start)))
// 			}
// 			fields = append(fields, slog.String("body", body.String()))
// 		} else {
// 			if start, ok := ctx.Value(contextHandlerStart).(time.Time); ok {
// 				fields = append(fields, slog.Duration("elapsed", time.Since(start)))
// 			}
// 			fields = append(fields, slog.Int64("contentLength", contentLength))
// 		}
// 		respMsg := fmt.Sprintf("response [%d] %s %s", statusCode, r.Method, r.RequestURI)
// 		lg.Info(respMsg, fields...)
// 	})
// }

// var defaultPool = &sync.Pool{
// 	New: func() interface{} {
// 		return bytes.NewBuffer(nil)
// 	},
// }

// func getBuffer() *bytes.Buffer {
// 	return defaultPool.Get().(*bytes.Buffer)
// }

// func putBuffer(b *bytes.Buffer) {
// 	b.Reset()
// 	defaultPool.Put(b)
// }
