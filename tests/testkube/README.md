# TestKube Tests

This directory contains TestKube workflow definitions for running tests in CI/CD environments. TestKube is a Kubernetes-native testing framework that orchestrates test execution within Kubernetes clusters.

## Overview

TestKube tests provide automated validation of the CloudZero Agent in CI/CD pipelines by:

- Running tests directly in Kubernetes clusters
- Providing standardized test workflows
- Integrating with existing CI/CD systems
- Offering centralized test management and reporting

## Test Definitions

### Agent Basic Test (`tests.yaml`)

Defines a basic health check workflow for the CloudZero Agent:

**What it tests:**

- **Health endpoint availability** - Verifies the webhook server health endpoint responds
- **Metrics endpoint accessibility** - Checks that metrics are available for collection
- **Service networking** - Validates internal Kubernetes service communication
- **Basic functionality** - Ensures core agent components are running

**Test workflow:**

1. Uses `curlimages/curl` container for HTTP testing
2. Tests webhook server health endpoint via HTTPS
3. Tests metrics endpoint via HTTP
4. Validates service DNS resolution within cluster

## Test Configuration

### Workflow Definition

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: agent-basic-test
  namespace: testkube
spec:
  pod:
    tolerations:
      - key: kubernetes.io/arch
        operator: Equal
        value: arm64
        effect: NoSchedule
  steps:
    - run:
        image: "curlimages/curl:8.13.0"
        shell: |
          curl -m 5 -k https://cz-agent-cloudzero-agent-webhook-server-svc.cz-agent.svc.cluster.local/healthz
          curl -m 5 http://cz-agent-cloudzero-state-metrics.cz-agent.svc.cluster.local:8080/metrics
```

### Key Features

- **Architecture tolerance** - Configured for arm64 nodes
- **Network testing** - Uses internal service DNS names
- **Timeout management** - 5-second timeout per request
- **SSL handling** - Bypasses SSL verification for self-signed certificates
- **Service discovery** - Tests Kubernetes service networking

## Integration with CI/CD

### TestKube Architecture

TestKube runs inside Kubernetes clusters and executes tests as Kubernetes jobs:

- Tests run in dedicated pods
- Results are collected and stored centrally
- Integrates with monitoring and alerting systems
- Provides web UI and API for test management

### Execution Context

These tests run in the same cluster where the CloudZero Agent is deployed:

- Access to internal service endpoints
- Real network conditions and policies
- Actual RBAC and security constraints
- Live metrics and health data

## Prerequisites

### TestKube Installation

TestKube must be installed in the target Kubernetes cluster:

```bash
# Install TestKube CLI
kubectl krew install testkube

# Or install via other methods - see TestKube documentation

# Install TestKube in cluster
testkube install
```

### Agent Deployment

The CloudZero Agent must be deployed and running:

- Webhook server must be accessible
- Metrics endpoint must be available
- Services must be properly configured
- Network policies must allow internal communication

## How to Run TestKube Tests

**IMPORTANT**: These tests run within Kubernetes clusters using the TestKube framework. They are NOT integrated with the main project Makefile - they run via TestKube's own orchestration.

### Via TestKube CLI

```bash
# Run the basic test workflow
testkube run workflow agent-basic-test

# Run with verbose output
testkube run workflow agent-basic-test -v

# Watch test execution in real-time
testkube run workflow agent-basic-test --watch
```

### Via TestKube Dashboard

1. Access TestKube web interface
2. Navigate to Workflows section
3. Select `agent-basic-test`
4. Click "Run" to execute

### Automated Execution

TestKube can be integrated with GitOps workflows:

- Triggered by deployment events
- Scheduled periodic execution
- Integrated with CI/CD pipelines
- Webhook-based triggers

## Test Results and Monitoring

### Expected Results

For a healthy CloudZero Agent deployment:

```bash
# Health endpoint should return 200 OK
curl -m 5 -k https://.../healthz
# Response: {"status":"ok"} or similar

# Metrics endpoint should return Prometheus metrics
curl -m 5 http://.../metrics
# Response: Prometheus format metrics data
```

### Result Interpretation

- **Success** - Both endpoints return successfully within timeout
- **Failure** - Network errors, timeouts, or unexpected responses
- **Partial** - One endpoint succeeds, other fails (indicates partial deployment issues)

### Monitoring Integration

TestKube results can integrate with:

- Prometheus metrics collection
- Grafana dashboard visualization
- Alert manager for failure notifications
- Slack/email notifications

## Troubleshooting

### Common Test Failures

1. **Connection timeout**:

   ```text
   curl: (28) Connection timed out after 5000 milliseconds
   ```

   - Check if agent pods are running and ready
   - Verify service names and namespaces are correct
   - Check network policies and firewall rules

2. **DNS resolution failure**:

   ```text
   curl: (6) Could not resolve host
   ```

   - Verify service names match actual deployed services
   - Check if services are in the correct namespace
   - Validate DNS is working in the cluster

3. **SSL certificate errors** (if `-k` flag removed):

   ```text
   curl: (60) SSL certificate problem: self signed certificate
   ```

   - Expected for self-signed certificates
   - Use `-k` flag to bypass certificate validation
   - Or configure proper certificates for production

4. **Permission denied**:
   ```text
   curl: (7) Failed to connect: Permission denied
   ```
   - Check RBAC permissions for TestKube service accounts
   - Verify network policies allow communication
   - Check if services are exposed on correct ports

### Debug Steps

```bash
# Check agent pod status
kubectl get pods -n cz-agent

# Check service endpoints
kubectl get endpoints -n cz-agent

# Test connectivity from within cluster
kubectl run debug --image=curlimages/curl --rm -it -- /bin/sh
# Then run curl commands manually

# Check TestKube test logs
testkube get execution <execution-id> --logs
```

## Extending TestKube Tests

### Adding New Test Workflows

Create additional YAML files in this directory:

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: agent-comprehensive-test
  namespace: testkube
spec:
  steps:
    - run:
        image: "appropriate-test-image"
        shell: |
          # More comprehensive test commands
          # Multiple validation steps
          # Performance testing
          # Error scenario testing
```

### Test Enhancement Ideas

- **Performance testing** - Load testing with multiple concurrent requests
- **Data validation** - Verify metrics content and format
- **Error scenarios** - Test behavior when services are unavailable
- **Security testing** - Validate authentication and authorization
- **Integration testing** - Test interaction with external services

### Best Practices

- Use appropriate timeout values for different test types
- Include both positive and negative test cases
- Test realistic failure scenarios
- Provide clear success/failure criteria
- Use descriptive test names and documentation

TestKube tests provide production-ready validation of the CloudZero Agent directly within Kubernetes environments, ensuring reliable operation in real deployment conditions.
