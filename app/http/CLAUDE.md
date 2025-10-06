# HTTP Utilities

## Purpose

Provides HTTP client utilities and middleware for external API integrations. Implements consistent HTTP communication patterns with timeout management and error handling.

## Components

**Client Utilities (`client/`):**

- `http.go` - HTTP client configuration and request handling
- `header.go` - HTTP header management utilities
- `query.go` - Query parameter processing
- `error.go` - HTTP error handling patterns

**Middleware (`middleware/`):**

- `middleware.go` - Request/response processing middleware

## Testing

```bash
make test GO_TEST_TARGET=./app/http/...
```
