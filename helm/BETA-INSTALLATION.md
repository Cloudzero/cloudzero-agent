# CloudZero Helm Charts Beta Installation Guide

This guide provides instructions on how to install and work with beta versions of CloudZero Helm charts. Beta versions may include new features and fixes that are not yet available in stable releases. Use them for testing and evaluation purposes.

## Adding the Beta Helm Repository

To access beta versions of the CloudZero Helm charts, you need to add the `beta` Helm repository:

```sh
helm repo add cloudzero-beta https://cloudzero.github.io/cloudzero-charts/beta
helm repo update
```

Here, we name the repository `cloudzero-beta`. Use this name in Helm commands when working with beta charts.

## Searching for Available Beta Versions

By default, Helm does not include pre-release (beta) versions when searching repositories. To list available beta versions of the `cloudzero-agent` chart, use the `--devel` flag:

```sh
helm search repo cloudzero-beta/cloudzero-agent --devel
```

**Example Output:**

```
NAME                            CHART VERSION   APP VERSION     DESCRIPTION
cloudzero-beta/cloudzero-agent  0.0.29-beta     v2.50.1         A chart for using Prometheus in agent mode to s...
```

The `--devel` flag includes development versions (alpha, beta, and release candidate) in the search results.

## Installing a Beta Version

There are two ways to install a beta version of the chart:

### Method 1: Using the `--devel` Flag

This method installs the latest beta version available.

```sh
helm install <RELEASE_NAME> cloudzero-beta/cloudzero-agent -n <NAMESPACE> --create-namespace -f configuration.example.yaml --devel
```

- The `--devel` flag allows Helm to consider beta versions when resolving the chart version.
- Replace `<RELEASE_NAME>` with the desired name for your Helm release.

### Method 2: Specifying the Exact Beta Version

If you want to install a specific beta version, specify it using the `--version` flag:

```sh
helm install <RELEASE_NAME> cloudzero-beta/cloudzero-agent -n <NAMESPACE> --create-namespace -f configuration.example.yaml --version <CHART_VERSION>
```

- Replace `<CHART_VERSION>` with the specific beta version (e.g., `1.0.0-beta`).
- This method does not require the `--devel` flag since you are explicitly specifying the version.

## Listing All Available Versions

To see all available versions of the `cloudzero-agent` chart, including both stable and beta versions, use the following command:

```sh
helm search repo cloudzero-beta/cloudzero-agent --versions --devel
```

**Example Output:**

```
NAME                            CHART VERSION   APP VERSION     DESCRIPTION
cloudzero-beta/cloudzero-agent  0.0.29-beta     v2.50.1         Cloudzero Agent with feature to s...
cloudzero-beta/cloudzero-agent  0.0.28-beta     v2.50.0         Cloudzero Agent with feature to u...
```

- The `--versions` flag lists all versions of the chart.
- The `--devel` flag includes pre-release versions in the list.

## Updating the Beta Chart Repository

Ensure that you have the latest versions of the beta charts by updating the repository:

```sh
helm repo update
```

## Additional Installation Instructions

Follow all other installation instructions as defined in the [README](./README.md), replacing `cloudzero` with `cloudzero-beta` in repository references when working with beta charts.

## Notes and Best Practices

- **Use in Testing Environments:** Beta versions are for testing and evaluation. Use them in non-production environments.
- **Specify Versions for Consistency:** When deploying to multiple environments, specify the exact chart version to ensure consistency.
- **Stay Informed of Changes:** Beta versions may introduce changes. Review release notes or change logs associated with the beta version you plan to use.
- **Remove Beta Repository When Not in Use:** If you no longer need access to beta charts, remove the repository to prevent accidental installation:

  ```sh
  helm repo remove cloudzero-beta
  ```

## Troubleshooting

- **Chart Not Found Error:** If you encounter an error like `no chart version found for cloudzero-agent-`, ensure you are using the `--devel` flag or specifying the exact beta version with `--version`.
- **Repository Not Updated:** If Helm doesn't recognize the latest beta charts, run `helm repo update` to refresh the repository cache.
- **Pre-release Versions Ignored:** Remember that Helm ignores pre-release versions by default. Always use `--devel` when searching or installing beta charts unless specifying the version.
