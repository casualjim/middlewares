package slogx

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockStringer struct {
	str string
}

func (m mockStringer) String() string {
	return m.str
}

func TestError(t *testing.T) {
	err := errors.New("test error")
	attr := Error(err)

	assert.Equal(t, "error", attr.Key)
	assert.Equal(t, slog.StringValue("test error"), attr.Value)
}

func TestByteString(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    []byte
		expected slog.Value
	}{
		{
			name:     "empty bytes",
			key:      "test",
			value:    []byte{},
			expected: slog.StringValue(""),
		},
		{
			name:     "ascii bytes",
			key:      "test",
			value:    []byte("hello"),
			expected: slog.StringValue("hello"),
		},
		{
			name:     "utf8 bytes",
			key:      "test",
			value:    []byte("hello 世界"),
			expected: slog.StringValue("hello 世界"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := ByteString(tt.key, tt.value)
			assert.Equal(t, tt.key, attr.Key)
			assert.Equal(t, tt.expected, attr.Value)
		})
	}
}

func TestStringer(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    mockStringer
		expected slog.Value
	}{
		{
			name:     "empty string",
			key:      "test",
			value:    mockStringer{str: ""},
			expected: slog.StringValue(""),
		},
		{
			name:     "simple string",
			key:      "test",
			value:    mockStringer{str: "hello"},
			expected: slog.StringValue("hello"),
		},
		{
			name:     "utf8 string",
			key:      "test",
			value:    mockStringer{str: "hello 世界"},
			expected: slog.StringValue("hello 世界"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Stringer(tt.key, tt.value)
			assert.Equal(t, tt.key, attr.Key)
			assert.Equal(t, tt.expected, attr.Value)
		})
	}
}

func TestLoggerName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected slog.Value
	}{
		{
			name:     "empty name",
			value:    "",
			expected: slog.StringValue(""),
		},
		{
			name:     "simple name",
			value:    "test-logger",
			expected: slog.StringValue("test-logger"),
		},
		{
			name:     "name with special chars",
			value:    "test.logger@domain",
			expected: slog.StringValue("test.logger@domain"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := LoggerName(tt.value)
			assert.Equal(t, KeyLoggerName, attr.Key)
			assert.Equal(t, tt.expected, attr.Value)
		})
	}
}

func TestKeyLoggerNameConstant(t *testing.T) {
	assert.Equal(t, "logger", KeyLoggerName)
}
