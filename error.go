package middlewares

import (
	"fmt"
	"net/http"
)

var _ error = (*httpError)(nil)

type httpError struct {
	statusCode int
	body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("[%d] %s", e.statusCode, e.body)
}

func Error(statusCode int, body string) error {
	return &httpError{statusCode: statusCode, body: body}
}

func ErrStatusCode(e error) int {
	if e == nil {
		return 0
	}
	if err, ok := e.(*httpError); ok {
		return err.statusCode
	}
	return 0
}

func ErrBody(e error) string {
	if e == nil {
		return ""
	}
	if err, ok := e.(*httpError); ok {
		return err.body
	}
	return ""
}

func IsBadRequest(e error) bool {
	return IsError(e, http.StatusBadRequest)
}

func IsUnauthorized(e error) bool {
	return IsError(e, http.StatusUnauthorized)
}

func IsForbidden(e error) bool {
	return IsError(e, http.StatusForbidden)
}

func IsNotFound(e error) bool {
	return IsError(e, http.StatusNotFound)
}

func IsServerError(e error) bool {
	if e == nil {
		return false
	}
	if err, ok := e.(*httpError); ok {
		return err.statusCode >= http.StatusInternalServerError
	}
	return false
}

func IsError(e error, code int) bool {
	if e == nil {
		return false
	}
	if err, ok := e.(*httpError); ok {
		return err.statusCode == code
	}
	return false
}
