# Security Policy

## Overview

CloudZero takes security seriously, especially for the CloudZero Agent, which is installed on customer infrastructure. We maintain a robust security posture through automated scanning, continuous monitoring, and proactive vulnerability management.

## Security Commitment

We are committed to:

- **Proactive Security**: Regularly scanning for vulnerabilities and maintaining up-to-date dependencies
- **Transparency**: Providing clear information about our security practices and vulnerability management
- **Responsive Disclosure**: Quickly addressing and communicating security issues
- **Best Practices**: Following industry standards for secure development and deployment

## Vulnerability Management

### Automated Security Scanning

We employ multiple layers of automated security scanning to maintain a strong security posture:

#### Container Image Scanning

All container images are automatically scanned for vulnerabilities using two complementary tools:

- **[Trivy](https://trivy.dev/latest/)**: Comprehensive vulnerability scanner that detects OS and library vulnerabilities
- **[Grype](https://github.com/anchore/grype)**: Fast vulnerability scanner with high accuracy

**Scanning Configuration**:

- **Severity Threshold**: High and Critical vulnerabilities trigger build failures
- **Scan Frequency**: Every build and monthly scheduled scans
- **Coverage**: OS packages, language dependencies, and configuration files

#### Code Security Analysis

- **[CodeQL](https://codeql.github.com/)**: Static analysis for security vulnerabilities in source code
- **[gosec](https://github.com/securecodewarrior/gosec)**: Go-specific security linting integrated into our CI/CD pipeline
- **[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)**: Go vulnerability checker that runs on every commit to identify known vulnerabilities in Go dependencies

### Dependency Management

We aggressively maintain up-to-date dependencies to minimize security risks:

#### Automated Updates

**[Dependabot](https://dependabot.com/)** automatically updates dependencies weekly, across multiple ecosystems:

- **Go Modules**
- **GitHub Actions**
- **Docker**
- **Helm Charts**
- **Node.js**

#### Update Process

1. **Automated Detection**: Dependabot identifies outdated dependencies
2. **Pull Request Creation**: Updates are proposed via pull requests
3. **Automated Testing**: All updates are tested through our CI/CD pipeline
4. **Security Validation**: Updated dependencies are scanned for vulnerabilities
5. **Manual Review**: Changes are reviewed by maintainers before merging

### Security Linting

Our development workflow includes comprehensive security linting:

- **gosec**: Detects common Go security issues
- **govulncheck**: Identifies known vulnerabilities in Go dependencies and runs on every commit
- **Static Analysis**: Additional security checks through staticcheck and other linters

## Vulnerability Reporting

### Reporting Security Issues

**Please do not report security vulnerabilities on the public GitHub issue tracker.**

Instead, email [security@cloudzero.com](mailto:security@cloudzero.com) with:

- **Description**: Clear description of the vulnerability
- **Impact**: Potential impact on users or systems
- **Reproduction**: Steps to reproduce the issue (if applicable)
- **Affected Versions**: Which versions are affected
- **Suggested Fix**: Any suggested remediation (optional)

### Response Process

1. **Acknowledgment**: You will receive an acknowledgment within 48 hours
2. **Investigation**: Our security team will investigate the reported issue
3. **Assessment**: We will assess the severity and impact
4. **Remediation**: We will develop and test a fix
5. **Disclosure**: We will coordinate disclosure with affected parties
6. **Release**: A security update will be released

### Responsible Disclosure

We follow responsible disclosure practices:

- **Timeline**: We aim to address critical issues within 30 days
- **Communication**: We will keep reporters informed of progress
- **Credit**: We will credit security researchers who report valid issues
- **Coordination**: We will coordinate with affected parties before public disclosure

## Security Best Practices for Users

### Keeping Your Deployment Secure

#### Regular Updates

**Always use the latest version of the CloudZero Agent Helm chart** to receive security updates. The latest releases can be found on the [CloudZero Agent Releases page](https://github.com/Cloudzero/cloudzero-agent/releases).

If you are using Helm directly:

```bash
# Check for updates
helm repo update cloudzero-agent

# Upgrade to latest version
helm upgrade cloudzero-agent cloudzero-agent/cloudzero-agent
```

**Important**: The Helm chart and container images are developed and released together as a coordinated unit. For security and stability reasons:

- **Version Synchronization**: Always use the container image version that matches your Helm chart version
- **Supported Combinations**: Only officially released chart/image combinations are supported
- **Security Implications**: Mismatched versions may cause unexpected issues or security issues
- **Compatibility**: Using incompatible versions can cause deployment failures or unexpected behavior

#### Security Configuration

1. **API Key Management**: Store API keys securely using Kubernetes secrets
2. **Network Security**: Configure network policies to restrict traffic
3. **RBAC**: Use least-privilege service accounts
4. **Resource Limits**: Set appropriate CPU and memory limits

#### Monitoring and Alerting

- Monitor pod logs for security-related events
- Set up alerts for failed security scans
- Regularly review access permissions
- Monitor for unusual network activity

### Security Considerations

#### Container Security

- **Base Images**: We use minimal base images to reduce attack surface
- **Non-Root Execution**: Containers run as non-root users when possible
- **Resource Limits**: All containers have defined resource limits
- **Security Contexts**: Proper security contexts are configured

#### Network Security

- **TLS Encryption**: All external communication uses TLS
- **Certificate Management**: Automatic certificate rotation and validation
- **Network Policies**: Configurable network policies for traffic control

#### Data Security

- **Encryption in Transit**: All data transmission is encrypted
- **Access Controls**: Role-based access control for all operations

## Security Metrics and Monitoring

### Continuous Monitoring

We continuously monitor our security posture through:

- **Vulnerability Scanning**: Automated scanning of all components
- **Dependency Analysis**: Regular analysis of dependency vulnerabilities
- **Security Metrics**: Tracking of security-related metrics
- **Incident Response**: Monitoring for security incidents

## Compliance and Standards

### Industry Standards

We follow industry best practices and standards:

- **OWASP**: Following OWASP security guidelines
- **CIS Benchmarks**: Implementing CIS security benchmarks
- **Kubernetes Security**: Following Kubernetes security best practices
- **Container Security**: Implementing container security best practices

## Contact Information

- **Security Issues**: [security@cloudzero.com](mailto:security@cloudzero.com)
- **General Support**: [support@cloudzero.com](mailto:support@cloudzero.com)
- **Documentation**: [CloudZero Docs](https://docs.cloudzero.com/)
