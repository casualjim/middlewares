# Go HTTP Middlewares

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](go.mod)

A collection of high-performance HTTP middlewares for Go web applications. These middlewares are designed to be composable and to work seamlessly with the standard `http.Handler` interface.

## Installation

```bash
go get github.com/casualjim/middlewares
```

## Available Middlewares

### Compression
- **CompressHandler** / **CompressHandlerLevel**: Compresses HTTP responses using gzip or deflate based on the client's Accept-Encoding header.

### Error Handling
- **RecoverRendered** / **Recover**: Catches panics in HTTP handlers and returns an appropriate error response.
- **Error**: Helper functions for working with HTTP errors.

### Content Negotiation
- **RequireJSONBody**: Validates that the request body contains valid JSON.
- **AllowMethods**: Restricts requests to specific HTTP methods.

### Response Helpers
- **JSON**: Helper for writing JSON responses.
- **JSONError**: Helper for writing JSON error responses.

### Logging
- **LoggingTransport**: Logs HTTP client requests and responses.
- **DebugDumpMiddleware**: Logs detailed HTTP server requests and responses.

### Caching Control
- **NoCache**: Prevents caching of HTTP responses.

### Profiling
- **NewProfiler**: Exposes the Go profiling endpoints at `/debug/pprof/`.

### Proxy Support
- **ProxyHeaders**: Properly handles headers set by reverse proxies.

## Usage Examples

### Basic Usage

```go
package main

import (
    "log/slog"
    "net/http"
    
    "github.com/casualjim/middlewares"
    "github.com/justinas/alice"
)

func main() {
    logger := slog.Default()
    
    // Create a middleware chain
    chain := alice.New(
        middlewares.Recover(logger),
        middlewares.CompressHandler,
        middlewares.NoCache,
        middlewares.ProxyHeaders,
    )
    
    // Your handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Respond with JSON
        middlewares.JSON(w, map[string]string{"message": "Hello, World!"})
    })
    
    // Use the middleware chain with your handler
    http.Handle("/", chain.Then(handler))
    http.ListenAndServe(":8080", nil)
}
```

### Error Handling and JSON Responses

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Validate that the request body is JSON
    body, err := middlewares.RequireJSONBody(w, r)
    if err != nil {
        middlewares.JSONError(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Process the body...
    
    // Respond with a successful JSON response
    middlewares.JSON(w, map[string]string{"status": "success"}, http.StatusOK)
}
```

### Method Restrictions

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Only allow GET and POST requests
    if err := middlewares.AllowMethods([]string{"GET", "POST"}, w, r); err != nil {
        return // AllowMethods already sets the appropriate response status and headers
    }
    
    // Continue processing...
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.