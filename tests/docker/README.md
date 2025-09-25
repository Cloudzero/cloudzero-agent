# Docker Test Utilities

This directory contains Dockerfiles and utilities for building test containers used in various test scenarios, particularly in smoke tests and integration tests.

## Overview

The Docker test utilities provide containerized environments for testing different components of the CloudZero Agent in isolation or as part of integrated test scenarios.

## Container Definitions

### Collector Container (`Dockerfile.collector`)

Builds a containerized version of the CloudZero Agent collector for testing:

**Purpose:**

- Test collector functionality in containerized environments
- Provide consistent collector behavior across test runs
- Enable collector testing without full Kubernetes deployment

**Usage:**

- Used in smoke tests for collector behavior validation
- Integration testing with mock data sources
- Performance testing under controlled conditions

### Controller Container (`Dockerfile.controller`)

Builds a test controller container for coordinating test scenarios:

**Purpose:**

- Orchestrate multi-container test scenarios
- Provide test coordination and timing
- Manage test data and state across containers

**Usage:**

- Smoke test coordination
- Multi-component integration testing
- Test result aggregation and reporting

### Shipper Container (`Dockerfile.shipper`)

Builds a containerized version of the CloudZero Agent shipper for testing:

**Purpose:**

- Test data shipping functionality in isolation
- Validate network communication and data formats
- Test error handling and retry logic

**Usage:**

- Smoke tests for data shipping workflows
- Network connectivity testing
- Data format and compression validation

### Remote Write Container (`Dockerfile.remotewrite`)

Builds a mock remote write endpoint for testing:

**Purpose:**

- Simulate CloudZero API endpoints
- Capture and validate shipped data
- Test authentication and network scenarios

**Usage:**

- Mock server for integration testing
- Data validation in smoke tests
- Network failure simulation

## Integration with Test Suites

### Smoke Tests

These containers are primarily used in smoke tests (`tests/smoke/`):

- Multi-container orchestration with testcontainers
- End-to-end data flow validation
- Network communication testing
- Error scenario simulation

### Container Orchestration

The containers work together in test scenarios:

1. **Collector** gathers test data
2. **Shipper** processes and sends data
3. **Remote Write** receives and validates data
4. **Controller** coordinates the entire flow

## Prerequisites

### Required Software

- **Docker** - For building and running containers
- **Docker Compose** (optional) - For multi-container scenarios
- **testcontainers-go** - Go library for container orchestration in tests

### Docker Setup

```bash
# Verify Docker is running
docker version

# Check available resources
docker system info
```

## How These Containers Are Used

**IMPORTANT**: These containers are NOT run directly. They are built and managed by the smoke test framework using testcontainers-go.

### Automatic Usage in Smoke Tests

```bash
# Smoke tests automatically build and use these containers
make test-smoke
```

The smoke test framework:

1. Builds containers from these Dockerfiles
2. Creates Docker networks for container communication
3. Starts containers in the correct order
4. Runs tests against the containerized environment
5. Cleans up containers automatically

### Manual Container Building (for development)

```bash
# Build individual containers for testing
cd tests/docker

# Build collector container
docker build -f Dockerfile.collector -t test-collector ../..

# Build shipper container
docker build -f Dockerfile.shipper -t test-shipper ../..

# Build remotewrite container
docker build -f Dockerfile.remotewrite -t test-remotewrite ../..

# Build controller container
docker build -f Dockerfile.controller -t test-controller ../..
```

## Container Configuration

### Build Context

All containers use the project root (`../..`) as build context to access:

- Source code from `app/` directory
- Configuration files
- Dependencies and build artifacts

### Environment Variables

Containers are configured through environment variables set by the test framework:

- API endpoints and credentials
- Test-specific configuration
- Network and timing parameters

### Volume Mounts

Containers may use volume mounts for:

- Shared test data
- Log output capture
- Configuration file sharing
- State synchronization between containers

## Development and Debugging

### Container Logs

During smoke tests, container logs are captured and displayed:

- Use `-v` flag with smoke tests for detailed output
- Logs help debug container startup and communication issues

### Interactive Testing

For development, you can run containers manually:

```bash
# Build and run collector container
docker build -f tests/docker/Dockerfile.collector -t test-collector .
docker run -it --rm test-collector

# Run with custom configuration
docker run -it --rm -e CONFIG_PATH=/test-config test-collector
```

### Network Testing

Test container communication:

```bash
# Create test network
docker network create test-network

# Run containers on same network
docker run --network test-network --name collector test-collector &
docker run --network test-network --name remotewrite test-remotewrite &

# Test connectivity
docker run --network test-network --rm curlimages/curl \
  curl http://remotewrite:8080/health
```

## Troubleshooting

### Common Issues

1. **Build failures**:

   ```bash
   # Check Docker daemon is running
   docker info

   # Clean build cache
   docker system prune

   # Build with no cache
   docker build --no-cache -f Dockerfile.collector .
   ```

2. **Container startup issues**:

   ```bash
   # Check container logs
   docker logs <container-id>

   # Run container interactively
   docker run -it --rm <image-name> /bin/sh
   ```

3. **Network connectivity**:

   ```bash
   # Check container networks
   docker network ls

   # Inspect network configuration
   docker network inspect <network-name>
   ```

### Debug Tips

- Use `docker run -it --rm <image> /bin/sh` to explore containers
- Check environment variables with `docker run <image> env`
- Use `docker logs -f <container>` to follow container logs in real-time
- Verify build context includes necessary files with `docker build --no-cache`

## Container Maintenance

### Regular Updates

- Keep base images updated for security
- Update dependencies in Dockerfiles
- Test containers with new Go versions
- Validate compatibility with new Docker versions

### Best Practices

- Use multi-stage builds for smaller images
- Minimize attack surface with minimal base images
- Set appropriate resource limits
- Use health checks for service containers
- Follow Docker security best practices

These Docker test utilities provide the foundation for comprehensive containerized testing of the CloudZero Agent, enabling realistic test scenarios without requiring full Kubernetes deployments.
