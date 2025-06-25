# cloudzero-scout CLI

A command-line interface for CloudZero Scout that automatically detects and retrieves cloud environment information. This tool is useful for debugging cloud environment detection, validating metadata service connectivity, and integration testing.

## Overview

The `cloudzero-scout` CLI provides an easy way to test and validate cloud environment auto-detection functionality. It supports multiple output formats and can be used for troubleshooting Scout integration issues in CloudZero Agent deployments.

## Installation

### Build from Source

```bash
# From the CloudZero Agent repository root
make build

# The binary will be available at:
# ./bin/cloudzero-scout
```

### Using Go Install

```bash
# Install directly from source
go install github.com/cloudzero/cloudzero-agent/app/functions/scout
```

## Usage

### Basic Commands

```bash
# Get full environment information (default command)
cloudzero-scout

# Only detect cloud provider
cloudzero-scout detect
```

### Command Reference

#### `cloudzero-scout` (default) or `cloudzero-scout info`

Retrieves complete cloud environment information including provider, region, and account ID.

**Examples:**

```bash
# Default JSON output
cloudzero-scout

# Table format output
cloudzero-scout --output table

# YAML format output
cloudzero-scout --output yaml

# With custom timeout
cloudzero-scout --timeout 30s

# Verbose output for debugging
cloudzero-scout --verbose
```

#### `cloudzero-scout detect`

Detects only the cloud provider without retrieving full metadata.

**Examples:**

```bash
# Detect cloud provider
cloudzero-scout detect

# Detect with table output
cloudzero-scout detect --output table
```

### Global Flags

| Flag        | Short | Default | Description                            |
| ----------- | ----- | ------- | -------------------------------------- |
| `--output`  | `-o`  | `json`  | Output format: `json`, `yaml`, `table` |
| `--timeout` | `-t`  | `10s`   | Timeout for metadata retrieval         |
| `--verbose` | `-v`  | `false` | Enable verbose output                  |

## Output Formats

### JSON (Default)

```bash
$ cloudzero-scout
{
  "CloudProvider": "aws",
  "Region": "us-east-1",
  "AccountID": "123456789012"
}
```

### YAML

```bash
$ cloudzero-scout --output yaml
cloudProvider: aws
region: us-east-1
accountId: "123456789012"
```

### Table

```bash
$ cloudzero-scout --output table
Cloud Provider : aws
Region         : us-east-1
Account ID     : 123456789012
```

## Examples

### AWS Environment

```bash
# Running on AWS EC2 instance
$ cloudzero-scout --output table --verbose
Initializing CloudZero Scout...
Timeout: 10s
Retrieving environment information...
Cloud Provider : aws
Region        : us-west-2
Account ID    : 975482786146
```

### Provider Detection Only

```bash
# Quick provider detection
$ cloudzero-scout detect
aws

# With JSON output
$ cloudzero-scout detect --output json
{
  "cloudProvider": "aws"
}
```

## Exit Codes

| Code | Description                                   |
| ---- | --------------------------------------------- |
| `0`  | Success - environment detected                |
| `1`  | Error - detection failed or invalid arguments |

## Related Documentation

- [Scout Package Documentation](../utils/scout/README.md) - Core Scout package API
- [CloudZero Agent Configuration](../../README.md) - Main project documentation
- [Build System](../../Makefile) - Build and development targets
