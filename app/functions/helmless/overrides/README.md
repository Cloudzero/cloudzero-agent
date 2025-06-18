# Overrides Package

The `overrides` package provides functionality for extracting configuration overrides by comparing configured values against default values from Helm charts.

## Overview

This package contains the core logic for identifying differences between configured values and default values, producing a minimal set of overrides that represent only the meaningful differences.

## Usage

```go
import "github.com/cloudzero/cloudzero-agent/app/functions/helmless/overrides"

// Create an extractor with optional exclude keys
extractor := overrides.NewExtractor("kubeStateMetrics", "excludedKey")

// Extract overrides
overrides := extractor.Extract(configuredValues, defaultValues)
```

## Key Features

### Extractor

The `Extractor` type handles the comparison logic and provides:

- **Configurable exclusions**: Specify keys to exclude from comparison
- **Deep comparison**: Recursively compares nested maps and arrays
- **Significance filtering**: Only includes values that are considered "significant"

### Significance Rules

A value is considered significant if it is:

- A non-empty string
- Any number (including zero)
- Any boolean value (true or false)
- A non-empty map with at least one significant value
- A non-empty array with at least one significant value

Values that are not significant:

- Empty strings
- `nil` values
- Empty maps
- Empty arrays
- Maps/arrays containing only insignificant values

## Example

```go
configured := map[string]interface{}{
    "replicas": 5,
    "image": "nginx:latest",
    "config": map[string]interface{}{
        "database": map[string]interface{}{
            "host": "prod.db.com",
            "port": 5432,
        },
    },
    "kubeStateMetrics": map[string]interface{}{
        "enabled": true,
    },
}

defaults := map[string]interface{}{
    "replicas": 3,
    "image": "nginx:latest",
    "config": map[string]interface{}{
        "database": map[string]interface{}{
            "host": "localhost",
            "port": 5432,
        },
    },
    "kubeStateMetrics": map[string]interface{}{
        "enabled": false,
    },
}

// Exclude kubeStateMetrics from comparison
extractor := overrides.NewExtractor("kubeStateMetrics")
result := extractor.Extract(configured, defaults)

// Result will be:
// {
//   "replicas": 5,
//   "config": {
//     "database": {
//       "host": "prod.db.com"
//     }
//   }
// }
```

## Testing

The package includes comprehensive tests covering:

- Basic extractor creation and configuration
- Simple and complex override scenarios
- Nested data structure handling
- Exclude key functionality
- Significance value determination
- Edge cases and type variations

Run tests with:

```bash
go test ./overrides
```
