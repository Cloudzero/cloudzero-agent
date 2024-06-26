# CloudZero Helm Charts

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/cloudzero/template-cloudzero-open-source.svg)

This repository contains helm charts for use by CloudZero users, which can be installed into any cloud-hosted Kubernetes cluster.

## Table of Contents

- [Current Helm Charts](#current-helm-charts)
- [Documentation](#documentation)
- [Installation](#installation)
- [Testing](#testing)
- [Contributing](#contributing)
- [Support + Feedback](#support--feedback)
- [Vulnerability Reporting](#vulnerability-reporting)
- [What is CloudZero?](#what-is-cloudzero)
- [License](#license)

## Current Helm Charts

- [cloudzero-agent](charts/cloudzero-agent/README.md) A chart for using Prometheus in agent mode to send cluster metrics to the CloudZero platform.

## Documentation

Detailed documentation of each helm chart should be available within each chart README.md. Other documentation that may be helpful:
- [CloudZero Docs](https://docs.cloudzero.com/) for general information on CloudZero.
- [Helm Documentation](https://helm.sh/) for information on what Helm is, and how it is used to install artifacts on Kubernetes clusters.
- [Kubernetes Documentation](https://kubernetes.io/docs/home/) for information on Kubernetes itself.

## Installation

The helm charts in this repository generally assume the use of Helm v3 for installation. More detailed installation instructions are located within the README of each chart. See the [official Helm documentation](https://helm.sh/docs/intro/install/) for instructions on installing Helm v3.

After the `helm` command is available, charts should be installable with the `install` or `upgrade` command:
```bash
helm upgrade --install <release-name> cloudzero/<chart-name>
```

Installation can also be managed by deployment tools such as ArgoCD or Spinnaker if desired, but installation instructions in this repository assume the use of the Helm CLI.

## Testing

Each helm chart should maintain its own [tests](https://helm.sh/docs/topics/chart_tests/). These tests should be executed with the command:
```bash
helm test <chart-name>
```

This is in addition to integration and load tests not managed in this repository.

## Contributing

We appreciate feedback and contribution to this repo! Before you get started, please see the following:

- [This repo's contribution guide](CONTRIBUTING.md)

## Support + Feedback

Contact support@cloudzero.com for usage, questions, specific cases. See the [CloudZero Docs](https://docs.cloudzero.com/) for general information on CloudZero.

## Vulnerability Reporting

Please do not report security vulnerabilities on the public GitHub issue tracker. Email [security@cloudzero.com](mailto:security@cloudzero.com) instead.

## What is CloudZero?

CloudZero is the only cloud cost intelligence platform that puts engineering in control by connecting technical decisions to business results.:

- [Cost Allocation And Tagging](https://www.cloudzero.com/tour/allocation) Organize and allocate cloud spend in new ways, increase tagging coverage, or work on showback.
- [Kubernetes Cost Visibility](https://www.cloudzero.com/tour/kubernetes) Understand your Kubernetes spend alongside total spend across containerized and non-containerized environments.
- [FinOps And Financial Reporting](https://www.cloudzero.com/tour/finops) Operationalize reporting on metrics such as cost per customer, COGS, gross margin. Forecast spend, reconcile invoices and easily investigate variance.
- [Engineering Accountability](https://www.cloudzero.com/tour/engineering) Foster a cost-conscious culture, where engineers understand spend, proactively consider cost, and get immediate feedback with fewer interruptions and faster and more efficient innovation.
- [Optimization And Reducing Waste](https://www.cloudzero.com/tour/optimization) Focus on immediately reducing spend by understanding where we have waste, inefficiencies, and discounting opportunities.

Learn more about [CloudZero](https://www.cloudzero.com/) on our website [www.cloudzero.com](https://www.cloudzero.com/)

## License

This project is licenced under the Apache 2.0 [LICENSE](LICENSE).
