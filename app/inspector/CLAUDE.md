# HTTP Response Inspector

## Purpose

Provides HTTP response inspection utilities for diagnosing CloudZero API integration issues. Enables response analysis and troubleshooting for external API communication.

## Components

- `inspector.go` - HTTP response inspection interface
- `response.go` - Response analysis and diagnostic utilities
- `query.go` - Query parameter analysis
- `common_headers.go` - HTTP header inspection and validation
- `status_code_inspectors.go` - Status code analysis patterns

## Testing

```bash
make test GO_TEST_TARGET=./app/inspector/...
```
