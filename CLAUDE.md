# CloudZero Agent Development Guide

## Repository Overview

This repository contains the complete CloudZero Agent ecosystem for Kubernetes integration with the CloudZero platform. It includes multiple applications, a comprehensive Helm chart, and supporting tools.

### Key Components

- **Applications**: Insights Controller, Collector, Shipper, Agent Validator, Agent (supports federated mode)
- **Helm Chart**: Complete Kubernetes deployment with automated testing and validation
- **Release Process**: Automated chart mirroring to `cloudzero-charts` repository on `develop` branch pushes
- **Testing**: Comprehensive test suites including unit, integration, helm schema, and smoke tests

### Development Commands

- `make build` - Build all binaries
- `make test` - Run unit tests
- `make helm-test` - Run helm validation tests
- `make package-build` - Build Docker images locally
- `TAG_VERSION=X.Y.Z make generate-changelog` - Generate/update changelog

### Release Process

1. Release notes must exist in `helm/docs/releases/{version}.md`
2. Manual "Prepare Release" GitHub workflow merges `develop` to `main`
3. Automatic chart mirroring syncs `helm/` to `cloudzero-charts` repository
4. Tags and GitHub releases are created automatically

## Changelog Format Guidelines

When generating or updating changelog files for CloudZero Agent releases, follow these formatting guidelines based on the existing changelog structure:

### File Structure

- **Filename**: `docs/releases/CHANGELOG-X.Y.md` (where X.Y is the minor version, e.g., 1.2)
- **Title**: `# CloudZero Agent X.Y.Z - Changelog`

### Content Structure

1. **Overview Section**
   - Brief summary of the release series
   - Highlight major themes or features introduced

2. **Major Features Section**
   - List new significant features with descriptive subsections
   - Use `###` for feature names
   - Include bullet points with detailed descriptions
   - Focus on user-facing capabilities and benefits

3. **Performance/Architecture Improvements**
   - Separate section for performance enhancements
   - Include monitoring, efficiency, and architectural changes
   - Use clear metrics when available (e.g., "every 1 minute previously 2 minutes")

4. **Configuration Changes**
   - New configuration options with default values
   - Breaking changes in configuration
   - API key management changes

5. **Bug Fixes Section**
   - Organize by version (e.g., "1.2.0 Fixes", "1.2.1 Fixes")
   - Use descriptive titles for each fix
   - Focus on user impact and resolution

6. **Breaking Changes**
   - Clearly list any breaking changes
   - Explain impact and migration requirements

7. **Security Section** (if applicable)
   - Vulnerability status
   - Security improvements

8. **Upgrade Path**
   - Provide clear upgrade instructions
   - Include helm commands with version placeholders

9. **Version History**
   - List all versions in the series with release dates
   - Brief description of each version's focus

### Formatting Guidelines

- Use consistent bullet point formatting (`-` for main points)
- Use `**Bold**` for emphasis on key terms and features
- Use code blocks for configuration examples and commands
- Use consistent date format: (YYYY-MM-DD)
- Group related changes under logical subsections
- Use present tense for describing features ("Provides", "Enables", "Supports")
- Focus on user benefits rather than technical implementation details

### Language Style

- Write user-focused descriptions
- Emphasize benefits and improvements
- Use clear, non-technical language where possible
- Be specific about improvements (include metrics when available)
- Maintain consistent tone across sections

### Example Entry Structure

```markdown
### Feature Name
- **Key Capability**: Description of what it does for users
- **Benefits**: How it helps users
- **Configuration**: Any relevant configuration details
```

This format ensures consistency across all CloudZero Agent changelog files and provides clear, actionable information for users upgrading or reviewing changes.