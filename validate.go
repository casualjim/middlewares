package middlewares

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

const (
	ContentTypeJSON string = "application/json"
	// https://github.com/ietf-wg-httpapi/mediatypes/blob/main/draft-ietf-httpapi-yaml-mediatypes.md
	ContentTypeYAML string = "application/yaml"
)

func RequireJSONBody(rw http.ResponseWriter, r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not read request body: %w", err)
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, ContentTypeJSON) {
		rw.WriteHeader(http.StatusUnsupportedMediaType)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", ct, ContentTypeJSON)
	}

	body = bytes.TrimSpace(body)

	if len(body) < 3 {
		rw.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("request body is required")
	}

	if body[0] != '{' && body[0] != '[' {
		rw.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("request body should contain a JSON Object `{}` or `[]`")
	}

	return body, nil
}

func AllowMethods(methods []string, rw http.ResponseWriter, r *http.Request) error {
	var matched bool
	for method := range slices.Values(methods) {
		if r.Method == method {
			matched = true
			break
		}
	}

	if !matched {
		joined := strings.Join(methods, ",")
		rw.Header().Add("Allow", joined)
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return fmt.Errorf("invalid method %s only %s requests are allowed", r.Method, joined)
	}
	return nil
}
