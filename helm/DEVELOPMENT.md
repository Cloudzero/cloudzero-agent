# Helm Chart Development Guide

> **Note**: This helm directory is mirrored to [github.com/cloud0/cloud0-charts](https://github.com/cloud0/cloud0-charts) and operates independently from the main project's Makefile system.

## Critical Template Modification Rules

**After ANY modification to Helm templates, you MUST:**

1. **Regenerate test manifests**: Update generated template outputs
2. **Review changes**: Check generated YAML matches expectations
3. **Verify correctness**: Ensure no unintended changes to Kubernetes resources

**This is not optional - template bugs can break deployments.**

## Template Development Best Practices

### Before Making Changes

**1. Complete Template Analysis:**

```bash
# Analyze the full template structure
helm template test-release . --show-only templates/target-template.yaml

# Identify all containers and their indices
helm template test-release . --show-only templates/target-template.yaml | grep -A 5 -B 5 "containers:"
```

**2. Container Index Verification:**

- **Simple templates**: Usually index [0] for main container
- **Complex templates**: May have multiple containers with specific indices
- **Init containers**: Separate indices from main containers
- **Conditional containers**: May not always be present

**3. Testing Strategy:**

- Use correct container indices in unittest tests
- Test both simple and complex template scenarios
- Verify semantic diff shows no unintended changes

### Template Structure Analysis

**Container Complexity Assessment:**

```bash
# Identify all containers in templates
grep -n "containers:" templates/*.yaml
grep -n "initContainers:" templates/*.yaml

# Count containers per template
helm template test-release . --show-only templates/my-template.yaml | grep -c "name:"

# Analyze container structure in detail
helm template test-release . --show-only templates/my-template.yaml | grep -A 10 -B 2 "containers:"
```

**Key Questions to Answer:**

- How many containers are in the template?
- Which containers use the resources being modified?
- Are there init containers that need separate consideration?
- What are the correct container indices for testing?
- Is this a complex multi-container template requiring careful index identification?

**Red Flags for Complex Templates:**

- Templates with `{{- if .Values.someCondition }}` blocks
- Templates with both `containers:` and `initContainers:`
- Templates with multiple container definitions
- Templates with conditional container rendering

### Schema Management

**Schema Dependencies:**

- Values schema files are auto-generated
- Schema regeneration is required after template changes
- Schema validation must pass before deployment

**Value Representation Strategy:**

```yaml
# CORRECT: Empty strings for "unset" values
resources:
  requests:
    memory: ""
    cpu: ""
  limits:
    memory: ""
    cpu: ""

# WRONG: Null values cause schema validation issues
resources:
  requests:
    memory: null
    cpu: null
```

### Backward Compatibility

**Fallback Logic Design:**

```yaml
# Example: Legacy precedence over components
{
  {
    - include "cloudzero-agent.generateResources" (include "cloudzero-agent.mergeStringOverwrite" (list
    (.Values.components.newComponent.resources | default (dict))
    (.Values.legacyComponent.resources | default (dict))
    ) | fromYaml) | nindent 10,
  },
}
```

**Key Considerations:**

- What's the intended precedence order for fallback logic?
- Should existing deployments continue to work without changes?
- Is this a breaking change or backward-compatible change?

## Local Template Testing

**For development and debugging, create local template tests:**

**Pattern**: Use `local-*` prefix for both overrides and output files:

```bash
# Create local override file
cat > tests/template/local-foo-overrides.yml <<EOF
apiKey: "not-a-real-api-key"

components:
  agent:
    replicas: 3
EOF

# Generate template output (process depends on chart testing setup)
helm template test-release . -f tests/template/local-foo-overrides.yml > tests/template/local-foo.yaml

# Review the generated output
cat tests/template/local-foo.yaml
```

**Benefits:**

- **Quick iteration**: Test specific overrides without affecting committed tests
- **Debugging**: See exactly what template changes produce in output
- **Experimentation**: Try different configurations safely

**Git ignored**: All `local-*` files in `tests/template/` are automatically ignored.

## HPA (Horizontal Pod Autoscaler) Patterns

**Custom metrics autoscaling:**

- Uses `czo_cost_metrics_shipping_progress` metric from collector
- No external dependencies (Prometheus Adapter, custom metrics server)
- Collector exposes Kubernetes custom metrics API v1beta1 at `/apis/custom.metrics.k8s.io/v1beta1/`

**Configuration pattern:**

```yaml
components:
  aggregator:
    autoscale: true

aggregator:
  scaling:
    minReplicas: 1
    maxReplicas: 10
    targetValue: "900m" # 90% of MaxRecords
```

**Template structure:**

- HPA template: `templates/aggregator-hpa.yaml`
- Conditional rendering with `if .Values.components.aggregator.autoscale`
- Custom metrics API reference in HPA spec
- Target value as resource quantity (e.g., "900m" for 0.9)

**Testing HPA templates:**

```bash
# Verify HPA configuration in deployed cluster
kubectl get hpa cloudzero-aggregator -n cloudzero-agent
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cloudzero-agent/pods/*/czo_cost_metrics_shipping_progress"
```

## Schema Testing

**Test file naming conventions:**

- `.pass.yaml` - tests that should succeed
- `.fail.yaml` - tests that should fail
- Use descriptive names: `global.empty.pass.yaml`

**Avoid redundant properties in test files:**

```bash
# Find files with auto-provided properties
git grep -P '^(cloudAccountId|clusterName|region|host|apiKey):' tests/schema/
```

**These properties are provided automatically by test framework:**

- `cloudAccountId`, `clusterName`, `region`, `host`, `apiKey`

## Implementation Checklist

**Before starting any Helm refactoring:**

- [ ] **Template Analysis**: Read entire template file, identify all containers
- [ ] **Schema Workflow**: Understand auto-generated files and regeneration process
- [ ] **Precedence Logic**: Clarify fallback strategy
- [ ] **Value Strategy**: Determine how to represent "unset" values
- [ ] **Testing Plan**: Map out all containers and their indices
- [ ] **Backward Compatibility**: Confirm existing deployments won't break

## Common Pitfalls to Avoid

- **Wrong Container Indices**: Always verify container indices in complex templates
- **Missing Schema Regeneration**: Always regenerate schema after template changes
- **Inconsistent Patterns**: Use established patterns consistently across components
- **Incomplete Testing**: Test all fallback scenarios, not just happy path
- **Poor Component Organization**: Group components logically, not by implementation details

## Integration with Main Project

While this helm chart operates independently, integration with the main CloudZero Agent project follows these patterns:

- **Image Repository**: `ghcr.io/cloudzero/cloudzero-agent/cloudzero-agent`
- **Current Version**: `1.2.7`
- **Multi-cluster Support**: Configurations for AWS EKS, Google GKE, and Azure AKS
- **Scout Integration**: Auto-detection of cloud provider metadata where supported
