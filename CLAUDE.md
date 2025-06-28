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

5. **Bug Fixes Section** - CRITICAL PLACEMENT RULES
   - **Section Header**: "## Reliability and Bug Fixes" followed by "### Major Bug Fixes Across X.Y.Z Series"
   - **Version Subsections**: Organize by version using "#### X.Y.Z Fixes" format
   - **New Versions**: Add new version fixes in chronological order WITHIN this existing section
   - **Location**: NEVER add content after "## Upgrade Path" or "## Version History"
   - **Format**: Use bullet points with "**Issue**: Description" format

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
   - UPDATE this section to include the new version being added

### CRITICAL SECTION PLACEMENT RULES

When adding content for a new version (e.g., 1.2.3):

1. **Major Features**: Add new subsections under existing "## Major Features" section using "### Feature Name" format
2. **Performance Improvements**: Add under existing "## Performance and Efficiency Improvements" section
3. **Bug Fixes**: Add under existing "### Major Bug Fixes Across 1.2.X Series" as "#### 1.2.3 Fixes"
4. **Version History**: Update the existing list to include the new version
5. **NEVER**: Add content after "## Upgrade Path" section
6. **NEVER**: Create duplicate section headers
7. **ALWAYS**: Maintain existing document structure and only INSERT within existing sections

### Content Classification Guidelines

**Major Features** (add to "## Major Features"):
- New standalone applications or services (e.g., collector, shipper, webhook)
- Significant user-facing capabilities
- Configuration simplification that reduces manual setup
- New integration support (cloud providers, tools)
- Auto-detection and zero-configuration capabilities

**IMPORTANT**: Distinguish between:
- **Standalone Applications**: collector, shipper, webhook, validator (deployed as separate containers)
- **Shared Packages/Libraries**: scout, utils, storage packages (used across applications)
- **Configuration Features**: Auto-detection capabilities enabled by shared packages

**Performance Improvements** (add to "## Performance and Efficiency Improvements"):
- Speed enhancements, memory optimizations
- Reduced resource usage
- Improved scalability

**Bug Fixes** (add to "### Major Bug Fixes Across 1.2.X Series"):
- Issue resolution, error fixes
- Template/validation improvements
- Certificate handling fixes

### Example of Correct Section Placement

```markdown
## Major Features

### Existing Feature 1
...existing content...

### Existing Feature 2  
...existing content...

### Configuration Automation  ← ADD NEW MAJOR FEATURES HERE
- **Cloud Provider Detection**: Automatic CSP metadata detection
- **Configuration Simplification**: Reduces manual configuration requirements
- **Multi-Cloud Support**: AWS and Google Cloud integration

## Performance and Efficiency Improvements

### Existing Performance Section
...existing content...

### Configuration Automation  ← ADD PERFORMANCE IMPROVEMENTS HERE
- **Reduced Manual Setup**: Scout eliminates need for manual region/account configuration
- **Faster Deployment**: Automatic environment detection speeds up installation

## Reliability and Bug Fixes

### Major Bug Fixes Across 1.2.X Series

#### 1.2.3 Fixes  ← ADD BUG FIXES HERE
- **Certificate Handling**: Fixed webhook certificate annotations
- **Template Validation**: Enhanced kubeconform integration

## Version History
- **1.2.3** (2025-06-27): Major release with Scout application and configuration automation ← UPDATE THIS
```

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