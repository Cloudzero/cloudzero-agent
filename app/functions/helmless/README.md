# Cloudzero Helmless

Helmless is a tool that compares configured values against default values from a
Helm chart, producing a minimal YAML file containing only the differences. This
is useful for understanding what values have been customized in a Helm
deployment of the CloudZero Agent for Kubernetes.

## Usage

```sh
helmless [flags]
```

### Flags

- `-c, --configured string` - Path to the configured values YAML file (default "configured-values.yaml")
- `-d, --defaults string` - Path to the default values YAML file (uses embedded defaults if not provided)
- `-o, --output string` - Path to write the output YAML file (default "-" for stdout)

### Example Workflows

#### Simplified Workflow (Using Embedded Defaults)

The tool now includes embedded default values from the Helm chart, so you can use it without manually extracting defaults:

1. Extract the current values from a deployed Helm release:

   ```sh
   kubectl -n cza get cm/cz-agent-helmless-cm -o jsonpath='{.data.values\.yaml}' > configured-values.yaml
   ```

2. Compare the values and generate a minimal overrides file:
   ```sh
   cloudzero-helmless \
       --configured configured-values.yaml \
       --output overrides.yaml
   ```

#### Traditional Workflow (Explicit Defaults)

If you need to use a different version of defaults or want explicit control:

1. Extract the current values from a deployed Helm release:

   ```sh
   kubectl -n cza get cm/cz-agent-helmless-cm -o jsonpath='{.data.values\.yaml}' > configured-values.yaml
   ```

2. Get the default values from the Helm chart:

   ```sh
   helm show values ./helm > default-values.yaml
   ```

3. Compare the values and generate a minimal overrides file:
   ```sh
   cloudzero-helmless \
       --configured configured-values.yaml \
       --defaults default-values.yaml \
       --output overrides.yaml
   ```

The resulting `overrides.yaml` will contain only the values that differ from the
defaults, making it easy to understand what has been customized.

## Getting Logs

When the helmless job runs as part of a Helm deployment, you can view its logs without needing to know the chart name by using the job's component label:

```sh
kubectl logs -l app.kubernetes.io/component=helmless -n <namespace> --tail=100
```

## Build System Integration

The embedded defaults are automatically generated during the build process by running `helm show values ./helm` and embedding the result into the binary. This ensures that the embedded defaults always match the current version of the Helm chart.

## Limitations

The `kubeStateMetrics` object is excluded from the comparison due to limitations
in the `helm show values` output
