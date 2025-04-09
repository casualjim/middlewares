package middlewares_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/casualjim/middlewares"
)

func TestHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		errFunc    func(error) bool
	}{
		{
			name:       "Test IsBadRequest",
			statusCode: 400,
			body:       "Bad Request",
			errFunc:    IsBadRequest,
		},
		{
			name:       "Test IsUnauthorized",
			statusCode: 401,
			body:       "Unauthorized",
			errFunc:    IsUnauthorized,
		},
		{
			name:       "Test IsForbidden",
			statusCode: 403,
			body:       "Forbidden",
			errFunc:    IsForbidden,
		},
		{
			name:       "Test IsNotFound",
			statusCode: 404,
			body:       "Not Found",
			errFunc:    IsNotFound,
		},
		{
			name:       "Test IsServerError",
			statusCode: 500,
			body:       "Internal Server Error",
			errFunc:    IsServerError,
		},
		{
			name:       "Test IsServerError",
			statusCode: 550,
			body:       "Internal Server Error",
			errFunc:    IsServerError,
		},
	}

	for tt := range slices.Values(tests) {
		t.Run(tt.name, func(t *testing.T) {
			err := Error(tt.statusCode, tt.body)
			assert.True(t, tt.errFunc(err))
			assert.Equal(t, tt.statusCode, ErrStatusCode(err))
			assert.Equal(t, tt.body, ErrBody(err))
		})
	}
}
