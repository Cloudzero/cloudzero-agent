# Common Issues FAQ: CloudZero Agent Installation Challenges

This document provides guidance on common challenges customers face when installing and configuring the CloudZero Agent Helm chart. Each section includes symptoms to watch for, diagnostic steps, and resolution strategies.

## Table of Contents

1. [Network Policy Issues](#network-policy-issues)
2. [Certificate Management Problems](#certificate-management-problems)
3. [Deployment Automation Challenges](#deployment-automation-challenges)
4. [Large Cluster Scaling Issues](#large-cluster-scaling-issues)
5. [Secret Management Problems](#secret-management-problems)
6. [Compliance and Security Requirements](#compliance-and-security-requirements)
7. [Resource Customization Challenges](#resource-customization-challenges)
8. [Image Management for Private Registries](#image-management-for-private-registries)

---

## Network Policy Issues

### Common Problems
- **Egress Restrictions**: Network policies blocking access to required external endpoints
- **S3 Bucket Access**: Blocked access to customer-specific S3 buckets
- **Internal Communication**: Namespace-to-namespace communication restrictions

### Symptoms to Watch For
- Agent pods failing to start or connect
- Timeout errors in logs
- Data not appearing in CloudZero platform
- Webhook validation failures

### Required Network Access
Customers must whitelist the following endpoints:
- `api.cloudzero.com` - CloudZero API endpoint
- `https://cz-live-container-analysis-<ORGID>.s3.amazonaws.com` - Customer-specific S3 bucket (where `<ORGID>` is the customer's Organization ID)

### Diagnostic Steps
1. Check pod logs for connection timeouts or DNS resolution failures
2. Test connectivity from within the cluster:
   ```bash
   kubectl run test-pod --image=curlimages/curl --rm -it -- curl -v https://api.cloudzero.com
   ```
3. Verify network policies allow egress to required endpoints
4. Check if internal namespace communication is blocked

### Resolution
- Work with customer's network team to whitelist required endpoints
- Review and update network policies to allow necessary egress traffic
- Ensure internal namespace communication is permitted for agent components

---

## Certificate Management Problems

### Common Problems
- **Service Mesh Interference**: Istio/Linkerd automatic mTLS injection conflicts with webhook certificates
- **Certificate Truncation**: Deployment automation (Flux) truncating certificate secrets
- **Self-Signed Certificate Issues**: Problems with init-cert job generated certificates

### Symptoms to Watch For
- Webhook validation failures
- Extra istio/linkerd containers in webhook pods (visible in `kubectl describe`)
- Certificate-related errors in validator logs
- Admission controller not responding

### Diagnostic Steps
1. **Check validator output** - Review validator logs from lifecycle hooks (visible in CloudZero Service Side DB)
2. **Test webhook communication**:
   ```bash
   # Deploy test pod and monitor webhook logs
   kubectl logs -f deployment/cloudzero-agent-webhook-server
   ```
3. **Test webhook endpoint directly**:
   ```bash
   # Create test ubuntu container in same namespace
   kubectl run test-ubuntu --image=ubuntu --rm -it -- bash
   # From within container, curl webhook endpoint with mock AdmissionReviewRequest
   ```
4. **Check for service mesh injection**:
   ```bash
   kubectl describe pod <webhook-pod-name>
   # Look for extra istio-proxy or linkerd containers
   ```

### Resolution
- For service mesh conflicts: Configure istio/linkerd to exclude webhook pods from automatic mTLS injection
- For certificate truncation: Review deployment automation configurations and ensure secrets are properly managed
- For self-signed certificate issues: Verify init-cert job completed successfully and secret was created properly

---

## Deployment Automation Challenges

### Common Problems
- **Template File Usage**: Customers using raw template files instead of helm template rendering
- **Complete values.yaml Override**: Copying entire values.yaml instead of minimal overrides
- **Upgrade Difficulties**: Problems during version upgrades due to excessive customization

### Symptoms to Watch For
- Frequent deployment failures during updates
- Customers reporting "template changes broke our deployment"
- Schema validation errors
- Upgrade issues between versions

### Best Practices for Customers

#### For Karpenter Users
- **Avoid**: Using raw template files directly (subject to change)
- **Recommended**: Use `helm template` to generate single rendered file:
  ```bash
  helm template cloudzero-agent cloudzero/cloudzero-agent -f values-override.yaml > cloudzero-agent-rendered.yaml
  ```
- Abstract the 3 primary variables in values-override.yaml

#### For ArgoCD/Flux Users
- **Avoid**: Copying entire values.yaml file
- **Recommended**: Only override necessary values in values-override.yaml
- Leverage built-in schema validation to prevent deployment errors

### Resolution
- Guide customers to minimal value overrides approach
- Emphasize using helm template for static deployments
- Explain schema validation benefits for preventing errors

---

## Large Cluster Scaling Issues

### Common Problems
- **High Memory Usage**: Agent consuming excessive memory in large clusters
- **Performance Degradation**: Slow metric collection and processing
- **Resource Contention**: Agent components competing for cluster resources

### Symptoms to Watch For
- High memory usage in `cloudzero-agent-server` container
- Slow metric collection or processing
- Pod restarts due to resource limits
- Performance issues in large clusters

### Scaling Solutions

#### Federated Mode (Daemonset Mode)
- **What it is**: Distributed agent deployment with sampling on each node
- **How it works**: Local sampling allows efficient scaling across large clusters
- **Configuration**: Enable federated flag in values to turn on daemonset mode
- **Benefits**: Reduces centralized processing load, improves scalability

#### Aggregator Scaling
- Increase replica sizes on aggregator to accommodate larger volume of remote writes
- Monitor aggregator performance and scale horizontally as needed

### Diagnostic Steps
1. Monitor memory usage: `kubectl top pods`
2. Check aggregator logs for performance issues
3. Review sizing guide in docs directory
4. Analyze cluster scale and workload patterns

### Resolution
- Enable federated/daemonset mode for large clusters
- Scale aggregator replicas based on cluster size
- Refer to sizing guide in docs directory for resource planning

---

## Secret Management Problems

### Common Problems
- **API Key Configuration**: Issues with Kubernetes secrets vs. direct values
- **External Secret Management**: Problems with third-party secret solutions
- **Secret Rotation**: Challenges with rotating API keys

### Supported Methods
- **Kubernetes Native Secrets**: Standard secret resources
- **Direct Values**: API key as direct value in configuration
- **External Secret Managers**: Various third-party solutions (AWS Secrets Manager, etc.)

### Configuration Requirements
For external secret management, ensure correct:
- Pre-existing secret name
- Secret file path
- Other specific settings per secret management solution

### Diagnostic Steps
1. **Validator Testing**: Validator fails install immediately if secret is bad
2. **Check validator logs**: Look for secret-related test failures
3. **Monitor shipper behavior**: Shipper holds data until good secret is provided

### Resolution
- Validator will report test failure in logs if secret is invalid
- Shipper supports dynamic secret rotation (no pod restart needed)
- Refer to AWS Secrets Manager guide in docs for specific implementations
- For other secret management solutions, ensure proper configuration per vendor requirements

---

## Compliance and Security Requirements

### Common Requirements
- **Source Code Review**: Customers want to inspect agent code
- **Security Scanning**: CVE scanning and security compliance validation
- **Testing Transparency**: Understanding of testing practices

### CloudZero Agent Security
- **Open Source**: Complete source code available at https://github.com/Cloudzero/cloudzero-agent
- **Automated Security**: Security scans and compliance concerns are automated
- **Transparency**: Full visibility into code, testing, and security practices

### Customer Guidance
Direct customers to GitHub repository for:
- Complete source code review
- Security scanning results
- Testing methodologies
- Compliance documentation

---

## Resource Customization Challenges

### Common Problems
- **Sizing Confusion**: Difficulty determining appropriate resource limits
- **Node Selector Issues**: Problems with node placement
- **Tolerations**: Challenges with pod scheduling constraints

### Available Resources
- **Sane Defaults**: Chart provides reasonable default resource limits
- **Sizing Guide**: Comprehensive guide available in docs directory
- **Configurable Values**: All resource settings exposed in values.yaml

### Scaling Considerations
- **Cluster Scale**: Resource needs depend on cluster size and workloads
- **Workload Patterns**: Different workload types may require different resources
- **Customer Responsibility**: DevOps teams must define appropriate limits for their environment

### Monitoring and Observability
Each service exposes endpoints for operations teams:
- **Health Checks**: `/healthz` endpoint for service health
- **Metrics**: `/metrics` endpoint for operational monitoring

### Resolution
- Direct customers to sizing guide in docs directory
- Emphasize that resource customization is environment-specific
- Highlight available health and metrics endpoints for monitoring

---

## Image Management for Private Registries

### Capability
- **Image Mirroring**: Customers can mirror CloudZero agent image to private registries
- **Single Image**: All agent utilities use a single image for simplified management
- **Configurable Values**: Image configuration exposed in chart values

### Limitations
- **Air-Gapped Systems**: Not supported - customers must have external connectivity
- **Support Scope**: Limited support for air-gapped environments

### Configuration
Customers can configure image settings in values.yaml:
```yaml
image:
  repository: <private-registry>/cloudzero-agent
  tag: <version>
  pullPolicy: IfNotPresent
```

### Resolution
- Guide customers to configure image values for private registries
- Clarify that air-gapped deployment is not supported
- Emphasize need for external connectivity to CloudZero services

---

## Quick Reference: First Steps for Common Issues

### Network Connectivity Problems
1. Check CloudZero Service Side DB for validator output
2. Test connectivity to api.cloudzero.com and customer S3 bucket
3. Review network policies and egress restrictions

### Certificate/Webhook Issues
1. Look for extra istio/linkerd containers in webhook pods
2. Check validator logs for certificate validation failures
3. Test webhook endpoint with mock requests

### Deployment Automation Problems
1. Verify customers are using minimal value overrides
2. Check for schema validation errors
3. Recommend helm template approach for static deployments

### Performance/Scale Issues
1. Monitor memory usage in cloudzero-agent-server container
2. Consider enabling federated/daemonset mode
3. Scale aggregator replicas as needed

### Secret Management Issues
1. Check validator logs for secret validation failures
2. Verify secret configuration matches chosen management method
3. Monitor shipper logs for authentication errors

---

## Escalation Guidelines

### When to Escalate
- Customer reports data not appearing in CloudZero platform after 10 minutes
- Persistent certificate issues after following troubleshooting steps
- Performance issues in large clusters after attempting scaling solutions

### Information to Gather
- Cluster size and workload characteristics
- Deployment method (ArgoCD, Flux, Karpenter, etc.)
- Network policy configurations
- Certificate management approach
- Error logs from validator, shipper, and webhook components

### Support Resources
- CloudZero Service Side DB for validator output
- Customer S3 bucket monitoring (visible within 10 minutes)
- GitHub repository for code review and security documentation