# Storage Package - Data Persistence

## Purpose

Implements Secondary Adapters for persistent data storage using Repository pattern with SQLite/GORM. Provides interfaces for metric storage, Kubernetes resource metadata, and operational state.

## Storage Types

### Database Storage (SQLite/GORM)

- **`sqlite/`** - SQLite driver and configuration
- **`repo/`** - Repository implementations using GORM
- **`core/`** - Base repository patterns and transaction management

### File Storage

- **`disk/`** - File-based storage for metrics and logs

## Core Repositories

- **Resource Store** - Kubernetes resource metadata persistence
- **Metric Store** - Prometheus metric data storage
- **Operational Store** - Agent configuration and runtime state

## Subdirectories

- **[core/](./core/)** - Base repository patterns and transaction management
- **[sqlite/](./sqlite/)** - SQLite database driver and configuration
- **[repo/](./repo/)** - Concrete repository implementations
- **[disk/](./disk/)** - File-based storage for streaming data

## Testing

```bash
# Test storage layer
make test GO_TEST_TARGET=./app/storage

# Test with real database (integration tests)
make test-integration GO_TEST_TARGET=./app/storage
```

## Architecture Role

**Secondary Adapter** - Implements persistence interfaces defined in `types/`. Domain layer depends on storage interfaces, not implementations.
