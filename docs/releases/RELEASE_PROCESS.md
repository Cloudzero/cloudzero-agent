# CloudZero Agent Release Process

This guide outlines the steps and best practices for managing releases of the CloudZero Agent. The process uses centralized changelog files and automated workflows to ensure consistency, quality, and proper chart mirroring to the cloudzero-charts repository.

## Overview

The CloudZero Agent release process has been streamlined to use centralized changelog files instead of individual release notes:

1. **Generate Changelog**
2. **Review Changelog**
3. **Trigger Manual Release Workflow**
4. **Automated Chart Mirroring**
5. **Publish Release**

---

## Step-by-Step Process

### 1. Generate Changelog

Use the automated changelog generation tool to create or update the changelog file:

```bash
# Generate changelog for version 1.2.3 (will update CHANGELOG-1.2.md)
TAG_VERSION=1.2.3 make generate-changelog
```

**What this does:**

- Analyzes git commits since the last release
- Extracts user-facing changes, bug fixes, and new features
- Updates or creates `docs/releases/CHANGELOG-X.Y.md` file
- Follows the established changelog format used across the project

**Changelog Location:**

- **Directory:** `docs/releases/`
- **Filename Format:** `CHANGELOG-X.Y.md` (e.g., `CHANGELOG-1.2.md`)
- **Content:** Version-specific sections within the changelog file

### 2. Review and Commit Changelog

Review the generated changelog for accuracy and completeness:

```bash
# Review the generated changelog
git diff docs/releases/CHANGELOG-*.md

# Make any necessary edits to improve clarity or add context
# Commit the changelog
git add docs/releases/CHANGELOG-*.md
git commit -m "Update changelog for version 1.2.3"
git push origin develop
```

**Review Guidelines:**

- Ensure user-facing language is clear and non-technical
- Verify all major features and breaking changes are captured
- Check that version sections are properly formatted
- Validate that grouped changes are logically organized

### 3. Trigger Manual Release Workflow

Navigate to GitHub Actions and trigger the release workflow:

- **Workflow:** "Manual Prepare Release"
- **Location:** `Actions > Manual Prepare Release`
- **Input Required:** Version number (e.g., `1.2.3`)

**What the workflow does:**

1. **Validates:** Confirms the required changelog file exists (`docs/releases/CHANGELOG-1.2.md`)
2. **Updates:** Helm chart version references and regenerates templates
3. **Merges:** `develop` branch into `main` with fast-forward merge
4. **Tags:** Creates git tag `v1.2.3`
5. **Extracts:** Release notes from the changelog file automatically
6. **Creates:** GitHub release (as draft) with extracted content

### 4. Automated Chart Mirroring

The chart mirroring process runs automatically on every push to `develop`:

**Mirror Workflow (`mirror-chart.yml`):**

- **Triggers:** Automatically on push to `develop` branch
- **Syncs:** `helm/` directory to `cloudzero-charts/charts/cloudzero-agent/`
- **Includes:** Changelog files are now synced to charts repository
- **Preserves:** Commit history and authorship information

**Chart Repository Structure:**

```
cloudzero-charts/
└── charts/
    └── cloudzero-agent/
        ├── templates/          # Helm templates
        ├── values.yaml         # Chart values
        ├── Chart.yaml          # Chart metadata
        └── docs/
            └── releases/       # Centralized changelog files (mirrored from docs/releases/)
                ├── CHANGELOG-1.0.md
                ├── CHANGELOG-1.1.md
                ├── CHANGELOG-1.2.md
                └── RELEASE_PROCESS.md
```

**Key Changes**:

- **Single Source**: `docs/releases/` is the authoritative location for all release documentation
- **No Duplication**: `helm/docs/releases/` legacy files are excluded from mirroring
- **Complete Sync**: Entire `docs/releases/` directory (including RELEASE_PROCESS.md) is mirrored to charts repo

### 5. Publish Release

After the release workflow completes:

1. **Review Draft Release:** Navigate to GitHub Releases and review the draft
2. **Verify Content:** Ensure release notes are properly extracted from changelog
3. **Publish Release:** Remove draft status to publish the release

**Post-Publication:**

- Container images are automatically built and published
- Charts repository is updated with latest changes
- Release notifications are sent to watchers

---

## Changelog Format and Guidelines

The CloudZero Agent uses centralized changelog files following a consistent format. The automated changelog generation tool analyzes git commits and creates well-structured changelogs.

### Changelog Structure

Changelog files follow this standardized format:

````markdown
# CloudZero Agent X.Y.Z - Changelog

## Overview

Brief summary of the release series and major themes.

## Major Features

### Feature Name

- **Key Capability**: Description of what it does for users
- **Benefits**: How it helps users
- **Configuration**: Any relevant configuration details

## Performance Enhancements

- **Improvement Description**: Include metrics when available
- **Enhanced Functionality**: Details about optimizations

## Bug Fixes Across X.Y.Z Series

### X.Y.0 Fixes

- **Issue Description**: Clear explanation of what was fixed
- **Resolution**: How the issue was resolved

## Breaking Changes

- **Change Description**: Impact and migration requirements

## Upgrade Path

```bash
helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -f configuration.yaml --version X.Y.Z
```
````

## Version History

- **X.Y.0** (YYYY-MM-DD): Initial release description
- **X.Y.1** (YYYY-MM-DD): Maintenance release description

````

### Automated Changelog Generation

The `make generate-changelog` command:

1. **Analyzes Commits**: Reviews git history since last release tag
2. **Extracts Changes**: Identifies user-facing changes, features, and fixes
3. **Categorizes Content**: Groups changes by type (features, fixes, improvements)
4. **Formats Output**: Follows existing changelog structure and style
5. **User-Focused Language**: Emphasizes benefits and impact to users

### Manual Changelog Enhancement

After generation, developers should:

1. **Review Accuracy**: Ensure all major changes are captured
2. **Improve Clarity**: Rewrite technical language for user-friendly descriptions
3. **Add Context**: Include configuration examples and upgrade guidance
4. **Verify Formatting**: Check markdown structure and section organization

---

## Workflow Details

### Release Workflow (`release-to-main.yml`)

The manual release workflow performs these automated steps:

1. **Validation Phase**:
   ```bash
   # Validates that the required changelog exists
   MINOR_VERSION=$(echo "1.2.3" | cut -d. -f1,2)
   test -f "docs/releases/CHANGELOG-${MINOR_VERSION}.md"
````

2. **Version Update Phase**:

   - Updates Helm chart image versions
   - Regenerates chart templates and tests
   - Commits version changes to `develop`

3. **Release Phase**:

   - Fast-forward merges `develop` into `main`
   - Creates git tag (e.g., `v1.2.3`)
   - Extracts release notes from changelog file
   - Creates GitHub draft release

4. **Release Notes Extraction**:
   ```bash
   # Automatically extracts version-specific content
   awk '/^## .*1.2.3/ { found=1; next } /^## / && found { exit } found { print }' CHANGELOG-1.2.md
   ```

### Chart Mirroring Workflow (`mirror-chart.yml`)

Automatically syncs changes to the charts repository:

1. **Trigger**: Every push to `develop` branch
2. **Sync Operations**:
   - Mirrors `helm/` directory to `cloudzero-charts/charts/cloudzero-agent/`
   - Copies changelog files to chart's `docs/releases/` directory
   - Preserves commit authorship and history
3. **Result**: Charts repository stays synchronized with latest changes

### Automation Benefits

- **Consistency**: Standardized changelog format across all releases
- **Efficiency**: Automated extraction eliminates manual copy-paste errors
- **Traceability**: Single source of truth for release information
- **Integration**: Seamless sync between repositories

---

## Best Practices

### Changelog Management

- **Single Source of Truth**: Use centralized changelog files instead of individual release notes
- **Automated Generation**: Leverage `make generate-changelog` for consistency
- **User-Focused Language**: Write for users, not developers - emphasize benefits and impact
- **Version Organization**: Group related changes by minor version series (1.2.x)
- **Regular Updates**: Update changelogs incrementally rather than at release time

### Release Coordination

- **Early Preparation**: Generate changelog files well before release deadlines
- **Stakeholder Review**: Allow time for team review of generated changelogs
- **Testing Integration**: Ensure release process includes full testing suite
- **Documentation Sync**: Verify charts repository receives updated documentation

### Quality Assurance

- **Validation**: Use automated workflow validation to catch issues early
- **Content Review**: Manually review automated changelog generation for accuracy
- **Format Consistency**: Follow established changelog structure and formatting
- **Link Verification**: Ensure all references and links are functional

---

## Troubleshooting

### Common Issues

**Changelog File Missing**:

```bash
# Error: test -f "docs/releases/CHANGELOG-1.2.md" fails
# Solution: Generate the changelog first
TAG_VERSION=1.2.3 make generate-changelog
```

**Release Notes Extraction Issues**:

```bash
# Problem: No content extracted from changelog
# Cause: Version section not found in changelog
# Solution: Ensure changelog has proper version headers (## 1.2.3)
```

**Chart Mirroring Delays**:

- Check that `develop` branch is up to date
- Verify mirror workflow completed successfully
- Confirm cloudzero-charts repository permissions

**Workflow Permissions**:

- Ensure `VERSION_BUMP_DEPLOY_KEY` secret is configured
- Verify `CLOUDZERO_CHARTS_DEPLOY_KEY` secret exists
- Check repository permissions for workflow execution

### Recovery Procedures

**Failed Release Workflow**:

1. Review workflow logs for specific error
2. Fix underlying issue (changelog, permissions, etc.)
3. Re-run workflow with same version number
4. Verify all steps complete successfully

**Missing Chart Updates**:

1. Manually trigger mirror workflow if needed
2. Verify chart repository has latest changes
3. Confirm changelog files are properly synced

---

## Migration from Legacy Process

### Key Changes from Previous Process

**Before (Legacy)**:

- Individual `helm/docs/releases/{version}.md` files
- Manual creation of release notes
- Separate chart and agent documentation

**After (Current)**:

- Centralized `docs/releases/CHANGELOG-{minor}.md` files
- Automated changelog generation with manual review
- Synchronized documentation across repositories

### Migration Steps for Existing Releases

1. **Consolidate Existing Notes**: Move individual release files into appropriate changelog files
2. **Update Workflows**: Ensure workflows reference changelog files instead of individual notes
3. **Clean Up Legacy Files**: Remove `helm/docs/releases/` directory to eliminate duplication
4. **Team Training**: Educate team on new `make generate-changelog` process
5. **Documentation Update**: Update all references to point to new process

### Cleanup Strategy

**Remove Legacy Release Files**:

```bash
# The helm/docs/releases/ directory can be safely removed
# All content has been consolidated into docs/releases/CHANGELOG-*.md files
rm -rf helm/docs/releases/

# Update any documentation that references the old location
# All chart documentation now sources from docs/releases/
```

**Benefits of Consolidation**:

- **Single Source of Truth**: Only `docs/releases/` contains release documentation
- **Reduced Maintenance**: No need to maintain duplicate files
- **Automatic Sync**: Charts repository gets complete release documentation
- **Simplified Workflow**: One location for all release-related files

---

## Integration with External Systems

### GitHub Releases

- Release content automatically extracted from changelog files
- Draft releases created for review before publication
- Tag creation and branch management fully automated

### CloudZero Charts Repository

- Automatic mirroring of helm chart and changelog files
- Preservation of commit history and authorship
- Seamless integration without manual intervention

### CI/CD Pipeline

- Integration with existing build and test processes
- Validation of changelog files before release
- Automated image building and publishing upon release

This updated process provides a streamlined, automated approach to releases while maintaining quality and consistency across the CloudZero Agent ecosystem.
