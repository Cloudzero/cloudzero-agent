# CloudZero Scout Auto Package

The Auto Scout package provides automatic cloud environment detection by trying multiple cloud provider scouts until one successfully detects the environment.

## Design

The auto package orchestrates multiple cloud provider scouts to provide automatic detection, rather than querying a specific cloud provider's metadata service directly.

## Features

- **Concurrent Detection**: Multiple scouts run concurrently for faster detection
- **Early Cancellation**: Once a provider is detected, other scouts are cancelled
- **Error Isolation**: Network errors in one scout don't prevent others from running
- **Caching**: Results are cached to avoid repeated detection calls
