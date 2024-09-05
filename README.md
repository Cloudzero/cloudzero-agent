# CloudZero Insights Controller

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-insights-controller.svg)

<img src="./docs/assets/deployment.png" alt="deployment" width="700">

The `CloudZero Insights Controller` provides telemetry to the CloudZero platform to enabling complex cost allocation and analysis.

## ⚡ Getting Started

The easiest way to get started it by using the [cloudzero-insights helm chart](https://github.com/Cloudzero/cloudzero-charts). 

### Installation

See the [Installation Guide](./INSTALL.md) for details.

### Configuration

See the [Configuration Guide](./CONFIGURATION.md) for details.

### Developer Quick Start

1. Build the image

    ```sh
    TAG=poc-simple make package
    ```

2. Deploy the admission controller

    ```sh
    make deploy-admission-controller
    ```

3. Monitor the logs in one console

    ```sh
    ./scripts/monitor-admission-contoller.sh
    ```

4. In another console, deploy a test app.

    ```sh
    make deploy-test-app
    ```
    > NOW - check out the logs in 3

###### Cleanup

```sh
make undeploy-admission-controller
make undeploy-test-app
```


## 🤝 How to Contribute

We appreciate feedback and contribution to this repo! Before you get started, please see the following:

- [This repo's contribution guide](CONTRIBUTING.md)

## 🤔 Support + Feedback

Contact support@cloudzero.com for usage, questions, specific cases. See the [CloudZero Docs](https://docs.cloudzero.com/) for general information on CloudZero.

## 🛡️ Vulnerability Reporting

Please do not report security vulnerabilities on the public GitHub issue tracker. Email [security@cloudzero.com](mailto:security@cloudzero.com) instead.

## ☁️ What is CloudZero?

CloudZero is the only cloud cost intelligence platform that puts engineering in control by connecting technical decisions to business results.:

- [Cost Allocation And Tagging](https://www.cloudzero.com/tour/allocation) Organize and allocate cloud spend in new ways, increase tagging coverage, or work on showback.
- [Kubernetes Cost Visibility](https://www.cloudzero.com/tour/kubernetes) Understand your Kubernetes spend alongside total spend across containerized and non-containerized environments.
- [FinOps And Financial Reporting](https://www.cloudzero.com/tour/finops) Operationalize reporting on metrics such as cost per customer, COGS, gross margin. Forecast spend, reconcile invoices and easily investigate variance.
- [Engineering Accountability](https://www.cloudzero.com/tour/engineering) Foster a cost-conscious culture, where engineers understand spend, proactively consider cost, and get immediate feedback with fewer interruptions and faster and more efficient innovation.
- [Optimization And Reducing Waste](https://www.cloudzero.com/tour/optimization) Focus on immediately reducing spend by understanding where we have waste, inefficiencies, and discounting opportunities.

Learn more about [CloudZero](https://www.cloudzero.com/) on our website [www.cloudzero.com](https://www.cloudzero.com/)

## 📜 License

This project is licenced under the Apache 2.0 [LICENSE](LICENSE).
