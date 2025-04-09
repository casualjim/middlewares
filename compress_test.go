package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/klauspost/compress/gzip"
)

// Taken from http://github.com/gorilla/handlers

const contentType = "text/plain; charset=utf-8"

func compressedRequest(w http.ResponseWriter, compression string) {
	CompressHandlerLevel(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(9*1024))
		w.Header().Set("Content-Type", contentType)
		for i := 0; i < 1024; i++ {
			_, _ = io.WriteString(w, "Gorilla!\n")
		}
	}), gzip.BestCompression).ServeHTTP(w, &http.Request{
		Method: "GET",
		Header: http.Header{
			"Accept-Encoding": []string{compression},
		},
	})
}

func TestCompressHandlerNoCompression(t *testing.T) {
	w := httptest.NewRecorder()
	compressedRequest(w, "")
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("wrong content encoding, got %q want %q", enc, "")
	}
	if ct := w.Header().Get("Content-Type"); ct != contentType {
		t.Errorf("wrong content type, got %q want %q", ct, contentType)
	}
	if w.Body.Len() != 1024*9 {
		t.Errorf("wrong len, got %d want %d", w.Body.Len(), 1024*9)
	}
	if l := w.Header().Get("Content-Length"); l != "9216" {
		t.Errorf("wrong content-length. got %q expected %d", l, 1024*9)
	}
}

func TestCompressHandlerGzip(t *testing.T) {
	w := httptest.NewRecorder()
	compressedRequest(w, "gzip")
	if w.Header().Get("Content-Encoding") != gzipa {
		t.Errorf("wrong content encoding, got %q want %q", w.Header().Get("Content-Encoding"), "gzip")
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Errorf("wrong content type, got %s want %s", w.Header().Get("Content-Type"), contentType)
	}
	if w.Body.Len() != 68 {
		t.Errorf("wrong len, got %d want %d", w.Body.Len(), 68)
	}
	if l := w.Header().Get("Content-Length"); l != "" {
		t.Errorf("wrong content-length. got %q expected %q", l, "")
	}
}

func TestCompressHandlerDeflate(t *testing.T) {
	w := httptest.NewRecorder()
	compressedRequest(w, "deflate")
	if w.Header().Get("Content-Encoding") != "deflate" {
		t.Fatalf("wrong content encoding, got %q want %q", w.Header().Get("Content-Encoding"), "deflate")
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Fatalf("wrong content type, got %s want %s", w.Header().Get("Content-Type"), contentType)
	}
	if w.Body.Len() != 50 {
		t.Fatalf("wrong len, got %d want %d", w.Body.Len(), 50)
	}
}

func TestCompressHandlerGzipDeflate(t *testing.T) {
	w := httptest.NewRecorder()
	compressedRequest(w, "gzip, deflate ")
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("wrong content encoding, got %q want %q", w.Header().Get("Content-Encoding"), "gzip")
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Fatalf("wrong content type, got %s want %s", w.Header().Get("Content-Type"), contentType)
	}
}

func TestCompressHandlerSSE(t *testing.T) {
	w := httptest.NewRecorder()
	CompressHandlerLevel(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: test\n\n")
	}), gzip.BestCompression).ServeHTTP(w, &http.Request{
		Method: "GET",
		Header: http.Header{
			"Accept-Encoding": []string{"gzip"},
		},
	})

	// Verify compression was skipped for SSE
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("expected no content encoding for SSE, got %q", enc)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("wrong content type, got %q want %q", ct, "text/event-stream")
	}
	expectedBody := "data: test\n\n"
	if w.Body.String() != expectedBody {
		t.Errorf("wrong body content, got %q want %q", w.Body.String(), expectedBody)
	}
}

func TestCompressHandlerChunked(t *testing.T) {
	w := httptest.NewRecorder()
	CompressHandlerLevel(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}
		_, _ = io.WriteString(w, "chunk1\n")
		flusher.Flush()
		_, _ = io.WriteString(w, "chunk2\n")
		flusher.Flush()
	}), gzip.BestCompression).ServeHTTP(w, &http.Request{
		Method: "GET",
		Header: http.Header{
			"Accept-Encoding": []string{"gzip"},
		},
	})

	// Verify compression was skipped for chunked response
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("expected no content encoding for chunked response, got %q", enc)
	}
	if te := w.Header().Get("Transfer-Encoding"); te != "chunked" {
		t.Errorf("expected chunked transfer encoding, got %q", te)
	}
	expectedBody := "chunk1\nchunk2\n"
	if w.Body.String() != expectedBody {
		t.Errorf("wrong body content, got %q want %q", w.Body.String(), expectedBody)
	}
}

func TestSetAcceptEncodingForPushOptionsWithoutHeaders(t *testing.T) {
	var opts *http.PushOptions
	opts = setAcceptEncodingForPushOptions(opts)

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Header)

	for k, v := range opts.Header {
		assert.Equal(t, "Accept-Encoding", k)
		assert.Len(t, v, 1)
		assert.Equal(t, "gzip", v[0])
	}

	opts = &http.PushOptions{}
	opts = setAcceptEncodingForPushOptions(opts)

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Header)

	for k, v := range opts.Header {
		assert.Equal(t, "Accept-Encoding", k)
		assert.Len(t, v, 1)
		assert.Equal(t, "gzip", v[0])
	}
}

// setAcceptEncodingForPushOptions sets "Accept-Encoding" : "gzip" for PushOptions without overriding existing headers.
func setAcceptEncodingForPushOptions(opts *http.PushOptions) *http.PushOptions {
	if opts == nil {
		opts = &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": []string{"gzip"},
			},
		}
		return opts
	}

	if opts.Header == nil {
		opts.Header = http.Header{
			"Accept-Encoding": []string{"gzip"},
		}
		return opts
	}

	if encoding := opts.Header.Get("Accept-Encoding"); encoding == "" {
		opts.Header.Add("Accept-Encoding", "gzip")
		return opts
	}

	return opts
}

func TestSetAcceptEncodingForPushOptionsWithHeaders(t *testing.T) {
	opts := &http.PushOptions{
		Header: http.Header{
			"User-Agent": []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.98 Safari/537.36"},
		},
	}
	opts = setAcceptEncodingForPushOptions(opts)

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Header)

	assert.Equal(t, "gzip", opts.Header.Get("Accept-Encoding"))
	assert.Equal(t, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.98 Safari/537.36", opts.Header.Get("User-Agent"))

	opts = &http.PushOptions{
		Header: http.Header{
			"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.98 Safari/537.36"},
			"Accept-Encoding": []string{"deflate"},
		},
	}
	opts = setAcceptEncodingForPushOptions(opts)

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Header)

	e, found := opts.Header["Accept-Encoding"]
	if !found {
		assert.Fail(t, "Missing Accept-Encoding header value")
	}
	assert.Len(t, e, 1)
	assert.Equal(t, "deflate", e[0])
	assert.Equal(t, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.98 Safari/537.36", opts.Header.Get("User-Agent"))
}
