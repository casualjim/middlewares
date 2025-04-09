package middlewares

import (
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/casualjim/middlewares/slogx"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"

	"github.com/felixge/httpsnoop"
)

const (
	deflate = "deflate"
	gzipa   = "gzip"
)

// gzipWriterPools stores a sync.Pool for each compression level for reuse of
// gzip.Writers. Use poolIndex to covert a compression level to an index into
// gzipWriterPools.
var gzipWriterPools [gzip.BestCompression - gzip.BestSpeed + 2]*sync.Pool

func init() {
	for i := gzip.BestSpeed; i <= gzip.BestCompression; i++ {
		addGzipLevelPool(i)
	}
	addGzipLevelPool(gzip.DefaultCompression)
}

// gzipPoolIndex maps a compression level to its index into gzipWriterPools. It
// assumes that level is a valid gzip compression level.
func gzipPoolIndex(level int) int {
	// gzip.DefaultCompression == -1, so we need to treat it special.
	if level == gzip.DefaultCompression {
		return gzip.BestCompression - gzip.BestSpeed + 1
	}
	return level - gzip.BestSpeed
}

func addGzipLevelPool(level int) {
	gzipWriterPools[gzipPoolIndex(level)] = &sync.Pool{
		New: func() interface{} {
			// NewWriterLevel only returns error on a bad level, we are guaranteeing
			// that this will be a valid level so it is okay to ignore the returned
			// error.
			w, _ := gzip.NewWriterLevel(nil, level)
			return w
		},
	}
}

// flateWriterPools stores a sync.Pool for each compression level for reuse of
// gzip.Writers. Use poolIndex to covert a compression level to an index into
// flateWriterPools.
var flateWriterPools [flate.BestCompression - flate.BestSpeed + 2]*sync.Pool

func init() {
	for i := flate.BestSpeed; i <= flate.BestCompression; i++ {
		addFlateLevelPool(i)
	}
	addFlateLevelPool(flate.DefaultCompression)
}

// flatePoolIndex maps a compression level to its index into flateWriterPools. It
// assumes that level is a valid flate compression level.
func flatePoolIndex(level int) int {
	// flate.DefaultCompression == -1, so we need to treat it special.
	if level == flate.DefaultCompression {
		return flate.BestCompression - flate.BestSpeed + 1
	}
	return level - flate.BestSpeed
}

func addFlateLevelPool(level int) {
	flateWriterPools[flatePoolIndex(level)] = &sync.Pool{
		New: func() interface{} {
			// NewWriterLevel only returns error on a bad level, we are guaranteeing
			// that this will be a valid level so it is okay to ignore the returned
			// error.
			w, _ := flate.NewWriter(nil, level)
			return w
		},
	}
}

// Adapted from http://github.com/gorilla/handlers
// Their middleware is greedy when it comes to implementing response writer methods
// this version uses httpsnoop to avoid expanding the interface and potentially break libraries
// that make assumptions based on those implementations

// CompressHandler gzip compresses HTTP responses for clients that support it
// via the 'Accept-Encoding' header.
//
// Compressing TLS traffic may leak the page contents to an attacker if the
// page contains user input: http://security.stackexchange.com/a/102015/12208
func CompressHandler(h http.Handler) http.Handler {
	return CompressHandlerLevel(h, gzip.DefaultCompression)
}

// CompressHandlerLevel gzip compresses HTTP responses with specified compression level
// for clients that support it via the 'Accept-Encoding' header.
//
// The compression level should be gzip.DefaultCompression, gzip.NoCompression,
// or any integer value between gzip.BestSpeed and gzip.BestCompression inclusive.
// gzip.DefaultCompression is used in case of invalid compression level.
func CompressHandlerLevel(h http.Handler, level int) http.Handler {
	if level < gzip.DefaultCompression || level > gzip.BestCompression {
		level = gzip.DefaultCompression
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	L:
		for enc := range slices.Values(strings.Split(r.Header.Get("Accept-Encoding"), ",")) {
			switch strings.TrimSpace(enc) {
			case gzipa:

				w.Header().Set("Content-Encoding", gzipa)
				w.Header().Add("Vary", "Accept-Encoding")

				index := gzipPoolIndex(level)
				gzw := gzipWriterPools[index].Get().(*gzip.Writer)
				gzw.Reset(w)
				defer func() {
					if err := gzw.Close(); err != nil {
						slog.Error("closing gzip writer", slogx.Error(err))
					}
					gzipWriterPools[index].Put(gzw)
				}()

				rrw := w
				var isStream bool
				w = httpsnoop.Wrap(w, httpsnoop.Hooks{
					WriteHeader: func(headerFunc httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
						return func(code int) {
							w.Header().Del("Content-Length")
							headerFunc(code)
						}
					},
					Write: func(_ httpsnoop.WriteFunc) httpsnoop.WriteFunc {
						return func(b []byte) (i int, e error) {
							h := w.Header()
							if h.Get("Content-Type") == "" {
								h.Set("Content-Type", http.DetectContentType(b))
							}

							if isStream || strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") || w.Header().Get("Transfer-Encoding") == "chunked" {
								isStream = true
								gzw.Reset(io.Discard)
								rrw.Header().Del("Content-Encoding")
								rrw.Header().Del("Vary")
								return rrw.Write(b)
							} else {
								h.Del("Content-Length")
							}
							return gzw.Write(b)
						}
					},
				})

				break L
			case deflate:
				w.Header().Set("Content-Encoding", deflate)
				w.Header().Add("Vary", "Accept-Encoding")

				index := flatePoolIndex(level)
				fw := flateWriterPools[index].Get().(*flate.Writer)
				fw.Reset(w)
				defer func() {
					if err := fw.Close(); err != nil {
						slog.Error("closing flate writer", slogx.Error(err))
					}
					flateWriterPools[index].Put(fw)
				}()

				rrw := w
				var isStream bool
				w = httpsnoop.Wrap(w, httpsnoop.Hooks{
					WriteHeader: func(headerFunc httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
						return func(code int) {
							w.Header().Del("Content-Length")
							headerFunc(code)
						}
					},
					Write: func(_ httpsnoop.WriteFunc) httpsnoop.WriteFunc {
						return func(b []byte) (i int, e error) {
							h := w.Header()
							if h.Get("Content-Type") == "" {
								h.Set("Content-Type", http.DetectContentType(b))
							}

							if isStream || strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") || w.Header().Get("Transfer-Encoding") == "chunked" {
								isStream = true
								fw.Reset(io.Discard)
								rrw.Header().Del("Content-Encoding")
								rrw.Header().Del("Vary")
								return rrw.Write(b)
							} else {
								h.Del("Content-Length")
							}
							return fw.Write(b)
						}
					},
					Push: func(httpsnoop.PushFunc) httpsnoop.PushFunc {
						return func(_ string, opts *http.PushOptions) error {
							if opts == nil {
								opts = &http.PushOptions{
									Header: http.Header{
										"Accept-Encoding": []string{"gzip"},
									},
								}
								return nil
							}

							if opts.Header == nil {
								opts.Header = http.Header{
									"Accept-Encoding": []string{"gzip"},
								}
								return nil
							}

							if encoding := opts.Header.Get("Accept-Encoding"); encoding == "" {
								opts.Header.Add("Accept-Encoding", "gzip")
								return nil
							}

							return nil
						}
					},
				})

				break L
			}
		}

		h.ServeHTTP(w, r)
	})
}
