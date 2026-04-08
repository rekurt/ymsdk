// Package middleware provides structured logging and HTTP debugging utilities
// for the Yandex Messenger Bot API SDK.
//
// The package includes three components:
//
//   - [ErrorLogger] — captures and logs API errors using [go.uber.org/zap],
//     with structured fields for error kind, HTTP status, and retry-after.
//   - [DebugLogger] — configurable debug logger with three verbosity levels
//     ([LogLevelError], [LogLevelInfo], [LogLevelDebug]) for development
//     and troubleshooting.
//   - [HTTPLogger] — wraps an [net/http.Client] to log raw HTTP request
//     and response bodies at debug level.
//
// Example setup with full HTTP inspection:
//
//	logger, _ := zap.NewDevelopment()
//	debugLog := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)
//	httpClient := middleware.NewHTTPLogger(http.DefaultClient, debugLog)
//	ymClient := ym.NewClientWithHTTP(cfg, httpClient)
package middleware
