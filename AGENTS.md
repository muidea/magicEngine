# AGENTS.md - magicEngine

This document provides guidelines for agentic coding assistants working in the magicEngine repository.

## Project Overview

magicEngine is a Go HTTP framework with TCP and SSE support. It provides middleware chains, routing, static file serving, and other web framework features.

## Quick Reference

### Essential Commands
```bash
# Build and test
go build ./...           # Build all packages
go test ./...            # Run all tests
go test ./http -v        # Run HTTP tests with verbose output
go test ./http -run TestPatternFilter  # Run specific test

# Code quality
go fmt ./...             # Format all code
go vet ./...             # Static analysis
golangci-lint run ./...  # Comprehensive linting (if installed)

# Quality check script
bash .agents/skills/go-refactor-pro/scripts/quality-check.sh
```

### Go Version
- **Minimum**: Go 1.24.0
- **Recommended**: Go 1.24.12+ (for security fixes)
- **Toolchain**: go1.24.11 (current)

## Testing

### Test Structure
- Tests are located in `*_test.go` files alongside the code they test
- Use standard Go testing package (`testing`)
- Test functions follow pattern: `TestFunctionName` or `TestType_Method`

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests for specific package with verbose output
go test ./http -v

# Run specific test function
go test ./http -run TestPatternFilter

# Run tests with coverage
go test ./http -cover
```

## Code Style Guidelines

### Imports
- Group imports: standard library, third-party, local packages
- Use absolute import paths for external packages
- Remove unused imports
- **Use `log/slog` for structured logging** (modern Go 1.21+)
- Example import structure:
```go
import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "strings"
    "sync"

    "github.com/muidea/magicCommon/foundation/util"
)
```

### Naming Conventions
- **Packages**: lowercase, single word, descriptive (e.g., `http`, `tcp`, `sse`)
- **Interfaces**: Use `er` suffix when appropriate (e.g., `Handler`, `Registry`)
- **Types**: PascalCase (e.g., `HTTPServer`, `RouteRegistry`)
- **Variables**: camelCase (e.g., `listenAddr`, `middlewareChains`)
- **Constants**: UPPER_SNAKE_CASE (e.g., `GET`, `POST`, `DefaultMaxFileSize`)
- **Private members**: lowercase (e.g., `httpServer`, `routeRegistry`)

### Error Handling
- Return errors from functions that can fail
- Use `errors.New` for simple errors, `fmt.Errorf` with `%w` for wrapping
- Define common errors as package-level variables in `errors.go`
- Handle errors at appropriate levels
- Use `errors.Join` for multiple errors (Go 1.20+)
- Example:
```go
// In http/errors.go
var (
    ErrURLNotFound = errors.New("the requested url was not found on this server")
    ErrMethodNotAllowed = errors.New("no matching http method found")
)

// Usage
func (s *EmbedStatic) findEmbedFile(filePath string) (*bytes.Reader, time.Time, error) {
    if filePath == "" {
        return nil, time.Time{}, ErrEmptyFilePath
    }
    // ...
}
```

### Type Definitions
- Define interfaces for abstraction
- Use structs for data containers
- Embed interfaces for composition
- Example interface definition:
```go
type HTTPServer interface {
    Use(handler MiddleWareHandler)
    Bind(routeRegistry RouteRegistry)
    Run()
}
```

### Middleware Pattern
- Middleware handlers implement `MiddleWareHandler` interface
- Use `RequestContext` for passing data between middleware
- Chain middleware using `ctx.Next()`
- Example middleware:
```go
type logger struct {
    serialNo int64
}

func (s *logger) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
    start := time.Now()
    ctx.Next()
    elapseVal := time.Since(start)
    // Log timing information
}
```

### HTTP Routing
- Routes implement `Route` interface
- Use `RouteRegistry` for managing routes
- Support HTTP methods: GET, POST, PUT, DELETE, HEAD, OPTIONS
- Pattern matching with wildcards (`**` for recursive matching)

### Concurrency
- Use `sync.Mutex` for protecting shared resources
- Use `sync.Map` for concurrent maps
- Use `atomic` operations for counters
- Example:
```go
type routeRegistry struct {
    routes          map[string]*routeItemSlice
    routesLock      sync.RWMutex
    currentApiVersion string
}
```

### Comments and Documentation
- Use GoDoc comments for exported types and functions
- Comments should explain "why" not "what"
- Keep comments concise and relevant
- Example:
```go
// NewHTTPServer creates a new HTTP server instance
// bindPort: port to listen on (e.g., "8080")
// enableShareStatic: whether to enable static file serving
func NewHTTPServer(bindPort string, enableShareStatic bool) HTTPServer {
    // ...
}
```

## Project Structure

```
magicEngine/
├── http/                    # HTTP framework package
│   ├── *.go                # Core HTTP implementation
│   └── *_test.go           # HTTP tests
├── tcp/                    # TCP framework package
│   ├── server.go
│   ├── client.go
│   └── endpoint.go
├── sse/                    # Server-Sent Events package
│   ├── server.go
│   └── client.go
├── example/                # Example implementations
│   ├── http/              # HTTP examples
│   ├── tcp/               # TCP examples
│   └── sse/               # SSE examples
├── go.mod                  # Go module definition
└── go.sum                  # Go dependencies
```

## Dependencies

- **Primary**: `github.com/muidea/magicCommon` (local replacement)
- **External**: Standard Go libraries only
- **Note**: The project uses a local replacement for magicCommon (`replace github.com/muidea/magicCommon => ../magicCommon`)
- **Logging**: Uses `log/slog` (Go 1.21+ standard library)

## Common Patterns

### 1. Factory Functions
- Use `NewTypeName` pattern for constructors
- Return interfaces, not concrete types
- Example: `NewHTTPServer`, `NewRouteRegistry`

### 2. Middleware Chains
- Middleware are executed in order of registration
- Each middleware calls `ctx.Next()` to continue chain
- Context can be modified and passed between middleware

### 3. Route Registration
- Routes are registered with method and pattern
- Middleware can be attached to routes
- API versioning supported

### 4. Static File Serving
- Supports embedded and filesystem static files
- Configurable root path and prefix
- MIME type detection

## Modern Go Features

### 1. Structured Logging with slog
- **Always use `log/slog` instead of `log` or `fmt` for logging**
- Use structured key-value pairs instead of formatted strings
- Example:
```go
// Before: log.Infof("listening on %s", s.listenAddr)
// After:
slog.Info("server listening", "addr", s.listenAddr)
slog.Error("server fatal error", "err", err)
```

### 2. Functional Options Pattern
- Use for constructors with multiple configuration parameters
- Example (see `http/embed_static.go`):
```go
type EmbedStaticOption func(*EmbedStatic)

func WithPrefixPath(path string) EmbedStaticOption {
    return func(es *EmbedStatic) {
        es.prefixPath = path
    }
}

func NewEmbedStatic(templateFS embed.FS, opts ...EmbedStaticOption) *EmbedStatic {
    es := &EmbedStatic{prefixPath: "/static"} // default
    for _, opt := range opts {
        opt(es)
    }
    return es
}
```

### 3. Type Safety
- **Never use strings as context keys** - define custom types
- Example:
```go
// Before: context.WithValue(ctx, "hello", "value")
// After:
type helloKey struct{}
context.WithValue(ctx, helloKey{}, "value")
```

### 4. Error Constants
- Define reusable error constants in `errors.go` files
- Use custom error types for additional context
- Example (see `http/errors.go`):
```go
var (
    ErrURLNotFound = errors.New("the requested url was not found on this server")
    ErrMethodNotAllowed = errors.New("no matching http method found")
)

type StaticError struct {
    Path string
    Err  error
}
```

## Development Workflow

1. **Make changes** to relevant `.go` files
2. **Run tests** to ensure functionality: `go test ./...`
3. **Build** to check compilation: `go build ./...`
4. **Test examples** to verify integration
5. **Run quality check**: `bash .agents/skills/go-refactor-pro/scripts/quality-check.sh`
6. **Commit changes** with descriptive messages

## Notes for Agents

- This is a framework library, not an application
- Focus on clean abstractions and interfaces
- Maintain backward compatibility for public APIs
- Follow Go conventions and idioms
- Test coverage is important for framework code
- Examples should demonstrate proper usage patterns

### Refactoring Guidelines
1. **Test First**: Check for `_test.go` files before refactoring
2. **DRY Principle**: Extract repeated code blocks (3+ occurrences)
3. **Modern Migration**: Use `slog` for logging, `errors.Join` for multiple errors
4. **Functional Options**: Refactor constructors with many parameters
5. **Type Safety**: Never use strings as context keys
6. **Error Constants**: Define reusable errors in `errors.go` files

## Troubleshooting

### Common Issues
1. **Missing dependencies**: Run `go mod download` and check local magicCommon
2. **Import errors**: Verify import paths and local replacements
3. **Test failures**: Check test environment and dependencies
4. **Build errors**: Ensure Go version compatibility (1.24.0+)

### Debugging Tips
- Use `go vet ./...` for static analysis
- Use `go fmt ./...` for formatting
- Check `go.mod` for correct dependencies
- Verify local replacement paths are correct

## Tools

You have access to a specialized Go refactoring skill. When performing complex refactoring tasks, use:
```bash
# Load the Go Refactor Pro skill
# This provides detailed workflows for safe refactoring, DRY principles,
# interface injection, functional options, and modern feature migration
```