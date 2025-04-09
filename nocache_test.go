package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoCache(t *testing.T) {
	rr := httptest.NewRecorder()
	ss := http.NewServeMux()
	ss.Handle("/", NoCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})))
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	ss.ServeHTTP(rr, r)

	for k, v := range noCacheHeaders {
		if rr.Header()[k][0] != v {
			t.Errorf("%s header not set by middleware.", k)
		}
	}
}
