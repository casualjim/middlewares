package middlewares

import "net/http"

// DefaultStack sets up the default middlewares
func DefaultStack(appInfo AppInfo, lgr Logger, orig http.Handler) http.Handler {
	recovery := NewRecovery(appInfo.Name, lgr, orig)
	gzip := Gzip(DefaultCompression, recovery)
	logger := NewAudit(appInfo, lgr, gzip)
	profiler := NewProfiler(logger)
	return NewHealthChecks(appInfo.BasePath, profiler)
}
