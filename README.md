# CloudZero Agent Validator

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-agent-validator.svg)

The `Cloud Agent Validator` is a simple CLI utility that performs various validation checks and send the results as telemetry to the CloudZero API.

The `Cloud Agent Validator` is a CLI utility designed to perform various validation checks and send the results as telemetry to the CloudZero API. It is intended to be used as part of the [pod lifecycle hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) in the [cloudzero-agent helm chart](https://github.com/Cloudzero/cloudzero-charts). During deployment, the validator runs checks and reports the lifecycle stage and test results to the API.

<img src="./docs/assets/deployment.png" alt="deployment" width="700">

This utility provides valuable information such as the `lifecycle stage`, enabling CloudZero to proactively engage with customers if an agent is experiencing issues reporting metrics. It also captures the versions of the chart and Prometheus agent, facilitating issue reproduction. Additionally, it reports check status results and error messages for failing checks, allowing Customer Success teams to quickly identify the root cause of any problems.

This repository also contains another tool, `cloudzero-agent-inspector`. Its primary purpose is to help diagnose errors and misconfigurations, and present them in a user-friendly and actionable way.

## ⚡ Getting Started

The easiest way to get started it by using the [cloudzero-agent helm chart](https://github.com/Cloudzero/cloudzero-charts). However if you'd like run the validator locally - this is also possible!

### Installation

If you are using docker, the easiest way to get started by running the following command:

```sh
docker run -it --rm ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator:latest cloudzero-agent-validator config generate
```

### Generate a configuration file

```sh
mkdir config
docker run -it --rm -v ./config:/config ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator:latest cloudzero-agent-validator config generate -f /config/myconfig.yml --account 123456789 --cluster my-cluster-name --region us-east-1
```

> You can now open `./config/myconfig.yml` and edit values as necessary for your cluster.

### Run a command locally (not in-pod)

In this example we are setting the conifguration value `credentials_file` to include our production CloudZero API Key. Run the following to create the file:

```sh
echo $CZ_API_KEY > config/credentials_file
```

Replace the value `credentials_file: /etc/config/prometheus/secrets/value` with `credentials_file: /config/credentials_file` in `config/myconfig.yml`

Now validate the configuration file:

```
docker run -it --rm -v ./config:/config ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator:latest cloudzero-agent-validator config validate -f /config/myconfig.yml
```

Then run a test to validate the API Token:

```sh
docker run -it --rm -v ./config:/config ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator:latest cloudzero-agent-validator diagnose run -f /config/myconfig.yml -check api_key_valid
```

## 🤝 How to Contribute

We appreciate feedback and contribution to this repo! Before you get started, please see the following:

- [This repo's contribution guide](CONTRIBUTING.md)

## Support + Feedback

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
