# Middleware

This package provides logging and HTTP instrumentation utilities for the ymsdk.

## Overview

The middleware package includes:
- **Error logging** with structured Zap integration
- **Debug logging** with configurable log levels
- **HTTP logging wrapper** for inspecting raw request/response bodies
- **Update logging helpers** for debugging message parsing issues

## Error Logging

Use `LogError` to log API and client errors with full context:

```go
import "github.com/rekurt/ymsdk/middleware"

logger, _ := zap.NewProduction()

// Log API errors
err := client.DoRequest(ctx, "GET", "/endpoint", nil)
if err != nil {
    middleware.LogError(logger, ctx, err, "GET", "/endpoint", params)
}
```

This logs the error kind, HTTP status, description, retry information, and request metadata.

## Debug Logging

The `DebugLogger` provides detailed HTTP inspection with configurable verbosity levels:

```go
import "github.com/rekurt/ymsdk/middleware"

logger, _ := zap.NewProduction()

// Create debug logger at DEBUG level
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)

// Logs are only written if level matches
debugLogger.LogDebug(ctx, "processing update")    // if level >= LogLevelDebug
debugLogger.LogWarning(ctx, "no message in update") // if level >= LogLevelWarn
debugLogger.LogInfo(ctx, "message received")       // if level >= LogLevelInfo
```

### Log Levels

- `LogLevelSilent` - No logging
- `LogLevelError` - Only errors
- `LogLevelWarn` - Warnings and errors
- `LogLevelInfo` - Info, warnings, and errors
- `LogLevelDebug` - All levels including detailed debug info

## HTTP Logging

Wrap your HTTP client to capture raw request/response bodies:

```go
import "github.com/rekurt/ymsdk/middleware"

logger, _ := zap.NewProduction()
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)

// Wrap the HTTP client
baseClient := &http.Client{Timeout: 15 * time.Second}
loggedClient := middleware.NewHTTPLogger(baseClient, debugLogger)

// Use with ymsdk
ymClient := ym.NewClientWithHTTP(cfg, loggedClient)
```

Output at DEBUG level:
```
DEBUG   HTTP Request
        method: POST
        url: https://botapi.messenger.yandex.net/bot/v1/messages/sendMessage
        body: {"chat_id":"...","text":"hello"}

DEBUG   HTTP Response
        status_code: 200
        body: {"ok":true,"result":{"id":12345,...}}
```

## Handling Updates Without Message

The SDK's `Update` struct has `Message` as an optional field. Not all updates include message data:

```go
err := cs.Updates.PollLoop(ctx, params, func(ctx context.Context, update ym.Update) error {
    if update.Message == nil {
        // Log this for debugging
        middleware.LogUpdateWithRawData(logger, ctx, update, rawJSON)
        return nil
    }

    // Process message
    return nil
})
```

The `LogUpdateWithRawData` function logs both the parsed update structure and the raw JSON for inspection.

## Common Debugging Scenarios

### 1. Inspecting Raw HTTP Payloads

Enable HTTP logging at DEBUG level to see exact bytes sent/received:

```go
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)
loggedClient := middleware.NewHTTPLogger(httpClient, debugLogger)
```

### 2. Tracking Updates Without Messages

```go
// Log update with raw data
middleware.LogUpdateWithRawData(logger, ctx, update, rawUpdateJSON)

// Or use LogUnparsedUpdate for completely malformed data
middleware.LogUnparsedUpdate(logger, ctx, []byte(`{bad json}`))
```

### 3. Conditional Logging

Use log levels to reduce noise:

```go
// Create debug logger at WARN level (no debug spam, but see warnings)
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelWarn)
```

## Best Practices

1. **Use structured logging**: Always use Zap field helpers (`zap.Int64`, `zap.String`, etc.)
2. **Don't log sensitive data**: The HTTP logger already skips Authorization headers
3. **Set appropriate level for environment**:
   - Production: `LogLevelWarn` or `LogLevelError`
   - Development: `LogLevelDebug`
4. **Check nil logger**: All logging functions check for nil logger first
