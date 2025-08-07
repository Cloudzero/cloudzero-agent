# Documentation Maintenance Guide

## Overview

This guide provides instructions for maintaining the comprehensive documentation system that has been established for the CloudZero Agent. The documentation follows a structured approach with both bottom-up (code-level) and top-down (architectural) coverage.

## Documentation Structure

### Code-Level Documentation (Phase 1 - Bottom-Up)
- **Package Documentation**: Every package has comprehensive overview documentation
- **Interface Documentation**: All public interfaces documented with contracts and examples
- **Function Documentation**: Parameters, returns, error conditions, and usage examples
- **Architecture Integration**: Code documentation references architectural patterns

### System Documentation (Phase 2 - Top-Down)  
- **ARCHITECTURE.md Files**: Layer-specific architectural documentation
- **Developer Guides**: Comprehensive development and contribution documentation
- **Operations Guides**: Deployment, monitoring, and maintenance documentation
- **API References**: Complete API documentation with examples and integration guides

## Maintenance Responsibilities

### Code Documentation Maintenance

#### When Adding New Components
1. **Package Documentation**: Add comprehensive package overview following established patterns
2. **Public APIs**: Document all public functions, interfaces, and types
3. **Examples**: Include usage examples for complex interfaces
4. **Architecture References**: Link to relevant architectural documentation

#### Documentation Standards
```go
// Package example provides comprehensive functionality for demonstrating documentation patterns.
//
// This package implements sophisticated business logic that integrates with the CloudZero
// agent architecture. It follows established patterns for:
//
//   - Domain-driven design with clear bounded contexts
//   - Repository pattern abstraction for data access
//   - Health monitoring integration with diagnostic providers
//   - Configuration management with environment detection
//
// Key features:
//   - Type-safe operations using Go generics
//   - Comprehensive error handling with context
//   - Performance optimization for high-throughput scenarios
//   - Security-first approach with input validation
//
// Usage:
//   service := example.NewService(repo, config)
//   if err := service.ProcessData(ctx, data); err != nil {
//       return fmt.Errorf("processing failed: %w", err)
//   }
package example
```

#### Function Documentation Template
```go
// ProcessData handles comprehensive data processing with validation and storage.
//
// This method implements the core business logic for data processing, including:
//   - Input validation and sanitization
//   - Business rule application
//   - Storage coordination with transactions
//   - Error handling and recovery
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - data: Input data to be processed (validated on input)
//
// Returns:
//   - error: Processing error or nil on success
//
// Error handling:
//   - ErrInvalidInput: Input data fails validation
//   - ErrStorageFailure: Database operation failed
//   - context.DeadlineExceeded: Operation timeout
//
// Example:
//   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//   defer cancel()
//   
//   if err := processor.ProcessData(ctx, inputData); err != nil {
//       log.Printf("Processing failed: %v", err)
//       return err
//   }
func ProcessData(ctx context.Context, data InputData) error
```

### Architectural Documentation Maintenance

#### When Adding New Layers or Components
1. **Update ARCHITECTURE.md**: Add new components to architectural diagrams and descriptions
2. **Component Integration**: Document how new components integrate with existing architecture  
3. **Pattern Documentation**: Update pattern documentation if new patterns are introduced
4. **Cross-References**: Update cross-references between architectural and code documentation

#### When Changing Architectural Patterns
1. **Pattern Migration**: Document migration from old to new patterns
2. **Backward Compatibility**: Document compatibility considerations
3. **Examples Update**: Update all examples to use new patterns
4. **Deprecation Notes**: Add deprecation notices for old patterns

### API Documentation Maintenance

#### When Adding New Endpoints
1. **Endpoint Documentation**: Add complete request/response documentation
2. **Authentication**: Document authentication and authorization requirements
3. **Error Responses**: Document all possible error conditions
4. **Rate Limiting**: Update rate limiting information
5. **Examples**: Provide comprehensive integration examples

#### When Changing APIs
1. **Version Documentation**: Document API version changes
2. **Breaking Changes**: Clearly mark breaking changes with migration guides
3. **Compatibility**: Document backward compatibility guarantees
4. **Client Updates**: Update client library examples

## Automated Maintenance Tasks

### Documentation Validation
```bash
#!/bin/bash
# docs/scripts/validate-documentation.sh

set -e

echo "Validating documentation completeness..."

# Check that all packages have documentation
find app -name "*.go" -path "*/testdata" -prune -o -name "*_test.go" -prune -o -print | \
while read -r file; do
    if ! grep -q "^// Package" "$file"; then
        echo "WARNING: $file missing package documentation"
    fi
done

# Check for broken internal links
find docs -name "*.md" | xargs grep -l "\[.*\](\.\./" | \
while read -r file; do
    echo "Checking links in $file..."
    # Check that referenced files exist
    grep -o "\[.*\](\.\.\/[^)]*)" "$file" | \
    sed 's/.*](\.\.\/\([^)]*\)).*/\1/' | \
    while read -r link; do
        if [[ ! -e "$link" && ! -e "$(dirname "$file")/../$link" ]]; then
            echo "ERROR: Broken link in $file: $link"
        fi
    done
done

echo "Documentation validation complete"
```

### Documentation Generation
```bash
#!/bin/bash
# docs/scripts/generate-api-docs.sh

set -e

echo "Generating API documentation..."

# Generate OpenAPI spec from code comments
go run ./tools/openapi-gen \
    --input-dirs ./app/handlers \
    --output-file docs/openapi.yaml

# Generate client library documentation
go run ./tools/client-doc-gen \
    --input ./app/handlers \
    --output docs/CLIENT_LIBRARIES.md

echo "API documentation generation complete"
```

## Documentation Review Process

### Pre-Commit Checks
1. **Spell Check**: Run spell checker on all documentation
2. **Link Validation**: Verify all internal and external links
3. **Format Consistency**: Check markdown formatting and structure
4. **Code Example Validation**: Ensure code examples compile and run

### Periodic Reviews
- **Monthly**: Review documentation completeness for new components
- **Quarterly**: Review architectural documentation for accuracy
- **Release Cycles**: Update API documentation for version changes
- **Annual**: Comprehensive documentation audit and cleanup

### Review Checklist
- [ ] New code has appropriate documentation coverage
- [ ] Architectural documentation reflects current system state
- [ ] API documentation includes all endpoints and changes
- [ ] Examples are current and functional
- [ ] Links are valid and point to correct resources
- [ ] Documentation follows established patterns and conventions

## Documentation Tools and Utilities

### Required Tools
```bash
# Install documentation maintenance tools
go install github.com/client9/misspell/cmd/misspell@latest
npm install -g markdownlint-cli
pip install mkdocs-material

# Validation script setup
chmod +x docs/scripts/validate-documentation.sh
chmod +x docs/scripts/generate-api-docs.sh
```

### Editor Configuration
```json
// .vscode/settings.json
{
    "markdownlint.config": {
        "MD013": false,  // Line length flexibility for technical docs
        "MD033": false,  // Allow HTML in markdown for diagrams
        "MD041": false   // Allow multiple H1 headers for complex docs
    },
    "cSpell.words": [
        "CloudZero",
        "Kubernetes", 
        "GORM",
        "Brotli",
        "healthz"
    ]
}
```

## Documentation Migration and Updates

### When Refactoring Code
1. **Update Package Documentation**: Reflect new package structure
2. **Update Examples**: Ensure examples use new APIs and patterns
3. **Add Migration Notes**: Document how to migrate from old to new APIs
4. **Deprecation Warnings**: Add warnings for deprecated functionality

### When Adding Features
1. **Feature Documentation**: Document new features in appropriate guides
2. **Configuration Updates**: Update configuration documentation
3. **API Documentation**: Add new endpoints to API reference
4. **Examples**: Add usage examples for new features

## Quality Metrics

### Documentation Coverage Metrics
- **Package Coverage**: Percentage of packages with comprehensive documentation
- **Function Coverage**: Percentage of public functions with documentation
- **API Coverage**: Percentage of endpoints with complete documentation
- **Example Coverage**: Number of documented features with working examples

### Quality Indicators
- **Link Health**: Percentage of documentation links that resolve correctly
- **Example Accuracy**: Percentage of code examples that compile and run
- **Freshness**: Time since last update for each documentation section
- **User Feedback**: Issues and questions related to documentation clarity

## Troubleshooting Documentation Issues

### Common Issues

#### Broken Links
```bash
# Find and fix broken internal links
find docs -name "*.md" -exec markdown-link-check {} \;
```

#### Outdated Examples
```bash
# Test all code examples in documentation
find docs -name "*.md" -exec grep -l "```go" {} \; | \
xargs -I {} bash -c 'echo "Testing examples in {}"; extract-and-test-go-examples.sh {}'
```

#### Missing Documentation
```bash
# Find packages missing documentation
find app -name "*.go" -not -path "*/testdata/*" -not -name "*_test.go" | \
while read file; do
    if ! grep -q "^// Package" "$file"; then
        echo "Missing documentation: $file"
    fi
done
```

This maintenance guide ensures that the comprehensive documentation system remains current, accurate, and valuable for all stakeholders working with the CloudZero Agent.