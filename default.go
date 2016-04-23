package middlewares

import "net/http"

// DefaultStack sets up the default middlewares
func DefaultStack(appInfo AppInfo, orig http.Handler) http.Handler {
	recovery := NewRecovery(appInfo.Name, orig)
	gzip := Gzip(DefaultCompression, recovery)
	logger := NewLogger(appInfo, gzip)
	profiler := NewProfiler(logger)
	return NewHealthChecks(appInfo.BasePath, profiler)
}
