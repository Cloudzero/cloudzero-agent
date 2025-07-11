# Makefile for backfiller testing

.PHONY: test-backfiller test-backfiller-cleanup test-backfiller-setup

# Check if kind and kubectl are installed
check-deps:
	@which kind > /dev/null || (echo "Error: kind is not installed. Please install kind first." && exit 1)
	@which kubectl > /dev/null || (echo "Error: kubectl is not installed. Please install kubectl first." && exit 1)

# Run the backfiller integration test
test-backfiller: check-deps
	@echo "Running backfiller integration test..."
	RUN_INTEGRATION_TESTS=true go test -v -tags=integration -timeout=5m ./...

# Manual cleanup (in case test fails to cleanup)
test-backfiller-cleanup:
	@echo "Cleaning up test resources..."
	-kind delete cluster --name cloudzero-backfiller-test
	-rm -rf /tmp/cloudzero-backfiller-test-*

# Check test status
test-backfiller-status:
	@echo "Checking test cluster status..."
	@echo "Kind clusters:"
	@kind get clusters || echo "No Kind clusters found"
	@echo "Test output directories:"
	@ls -la /tmp/cloudzero-backfiller-test-* 2>/dev/null || echo "No test output directories found"

# Setup test dependencies (optional)
test-backfiller-setup: check-deps
	@echo "Test dependencies are available"

# Run test with verbose output and keep cluster for debugging
test-backfiller-debug: check-deps
	@echo "Running backfiller integration test in debug mode..."
	@echo "Note: First run may take 5-10 minutes to download Kind node image..."
	@if [ -z "$$CLOUDZERO_DEV_API_KEY" ] && [ -n "$$CZ_DEV_API_TOKEN" ]; then \
		echo "Using CZ_DEV_API_TOKEN as CLOUDZERO_DEV_API_KEY..."; \
		export CLOUDZERO_DEV_API_KEY="$$CZ_DEV_API_TOKEN"; \
	fi; \
	RUN_INTEGRATION_TESTS=true CLOUDZERO_DEV_API_KEY="$${CLOUDZERO_DEV_API_KEY:-$$CZ_DEV_API_TOKEN}" go test -v -tags=integration -timeout=15m ./... || true
	@echo "Cluster left running for debugging. Use 'make test-backfiller-cleanup' to clean up."

help:
	@echo "Available targets:"
	@echo "  test-backfiller          - Run the backfiller integration test"
	@echo "  test-backfiller-cleanup  - Clean up test resources"
	@echo "  test-backfiller-setup    - Check test dependencies"
	@echo "  test-backfiller-debug    - Run test in debug mode (keep cluster)"
	@echo "  test-backfiller-status   - Check test cluster and output status"
	@echo "  help                     - Show this help message"