# Proof of Value Quick Start

This PoV is designed to allow one to reproduce the results quickly, while still keeping the effort low to deliver the discovery findings.

## Prerequisites

Start by making sure you have the following tools installed:

* [aws cli](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
* [aws sam](https://github.com/localstack/aws-sam-cli-local)
* [make](https://formulae.brew.sh/formula/make)
* [golang 1.23](https://go.dev/doc/install)

## Quick Start

Most things have been added to the [Makefile](./Makefile). Running `make` will provide you with a menu of the targets.

1. Authenticate to the Research AWS Account

    ```sh
    use_profile cz-research.Engineering
    ```

2. Build and deploy the project

    ```sh
    make build deploy
    ```

3. Pull down files from S3 (Alfa for example to a local directory), then use `scripts/sendfiles.sh` to sent them to your API. Note you will need to update the endpoint to match your deployment API Gateway ID.

---

Need help? Reach out to `joe.barnett` on slack. Happy coding!
