# Helm Overrides Extractor

A tool that compares configured values against default values from a Helm chart,
producing a minimal YAML file containing only the differences. This is useful
for understanding what values have been customized in a Helm deployment of the
CloudZero Agent for Kubernetes.

## Usage

```sh
helm-overrides-extractor [flags]
```

### Flags

- `-c, --configured string` - Path to the configured values YAML file (default "configured-values.yaml")
- `-d, --defaults string` - Path to the default values YAML file (default "default-values.yaml")
- `-o, --output string` - Path to write the output YAML file (default "-" for stdout)

### Example Workflow

1. Extract the current values from a deployed Helm release:
   ```sh
   kubectl -n cza get cm/cz-agent-config-values -o jsonpath='{.data.values\.yaml}' > configured-values.yaml
   ```

2. Get the default values from the Helm chart:
   ```sh
   helm show values ./helm > default-values.yaml
   ```

3. Compare the values and generate a minimal overrides file:
   ```sh
   helm-overrides-extractor \
       -configured configured-values.yaml \
       -defaults default-values.yaml \
       -output overrides.yaml
   ```

The resulting `overrides.yaml` will contain only the values that differ from the
defaults, making it easy to understand what has been customized.

## Limitations

The `kubeStateMetrics` object is excluded from the comparison due to limitations
in the `helm show values` output
