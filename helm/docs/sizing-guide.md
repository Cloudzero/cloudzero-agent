### Prometheus Agent Deployment Sizing Guide

#### Introduction

This guide outlines the recommended settings for deploying the Prometheus agent using the CloudZero Helm chart. It includes instructions on how to configure memory and CPU resources based on the size of your cluster.

#### Baseline Memory Requirements

- **Base Memory:** 512Mi
- **Base Memory Limit:** 1024Mi
- **Additional Memory:** 0.75Gi per 100 nodes in the cluster

#### Sizing Calculation

It is recommended to consider the shape and size of your prometheus cluster when setting resource memory limits for the prometheus agent. To calculate the memory requirements for your cluster, one can use the following formula:

![sizing formula](./assets/sizing-formula.png)

> This guide uses a basic formula based on number of nodes in the cluster. Please note, your mileage may vary if you have:
>
> - Very large machines, with a large number of pods
> - High churn pods or jobs. Each pod started triggers allocation of a memory for that pods metrics cache in the agent's memory. If the pod restarts, a new cache is created for the new pod instance. [More details on the cache can be found in the prometheus documentation.](https://prometheus.io/docs/practices/remote_write/). This cache is maintained for 2 hours to handle failure recovery of remote writes.

#### Sample values-override.yml Configuration

Create a `values-override.yml` file or edit the default `value.yml` file with the following content to configure the resource limits and requests for your Prometheus agent deployment. Replace `<CALCULATED_MEMORY_LIMIT>` with the actual number of nodes in your cluster:

```yaml
server:
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: "<CALCULATED_MEMORY_LIMIT>"
```

When using Helm, you can provide specific values in a separate `values-override.yml` file to override the defaults specified in the original `values.yml`. This approach allows you to override only the necessary values rather than providing the entire block.

#### Example Configuration for 200 Nodes

Calculate the memory limit based on the number of nodes, for example 200 nodes, the configuration would be:

![Example](./assets/sizing-formula-eg.png)

Example `values-override.yml`:

```yaml
server:
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: 2048Mi
```

This file only includes the overrides for the server resources size limit.

By following these instructions, you can ensure your Prometheus agent is properly sized to handle your cluster's load, preventing potential memory issues and ensuring smooth operation.
