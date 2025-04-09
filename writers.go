package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/goccy/go-json"

	"github.com/casualjim/middlewares/slogx"
)

func JSONError(w http.ResponseWriter, error string, code int, headers ...http.Header) {
	for header := range slices.Values(headers) {
		for k, v := range header {
			for val := range slices.Values(v) {
				w.Header().Add(k, val)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if code > 0 {
		w.WriteHeader(code)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintf(w, `{"message":%q,"code":%d}`, error, code)
}

func JSON[T any](w http.ResponseWriter, data T, code ...int) {
	status := http.StatusOK
	if len(code) > 0 {
		status = code[0]
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("write json body to response", slogx.Error(err))
	}
}
