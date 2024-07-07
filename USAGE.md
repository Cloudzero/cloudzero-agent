# Cloudzero Agent Validator User Guide

The Cloudzero Agent Validator is a tool designed to validate Cloudzero-Agent chart deployment, and perform various sanity checks. Data collected is used to set the status of the cluster in the dashboard, allowing immediate feedback to the user with the Cloudzero Dashboard.

The following guide outlines the built in commands available in the Cloudzero Agent Validator.

## Usage

To use the Cloudzero Agent Validator, follow the syntax below:

```sh
cloudzero-agent-validator [global options] command [sub-command options]
```

---

## Commands

The Cloudzero Agent Validator supports the following top-level commands:

### `config`
The `config` command provides configuration utility commands. Use the following syntax:

```sh
cloudzero-agent-validator config [sub-command] [command options]
```

#### Sub-commands

##### `generate`

The `generate` command generates a generic config file. Use the following syntax:

```
cloudzero-agent-validator config generate -account <CloudAccountID> -cluster <ClusterName> -region <Deployment Region>
```

##### `validate`

The `validate` command validates the config file. Use the following syntax:

```
cloudzero-agent-validator config validate --config <path to configuration file>
```

### `diagnose`

The `diagnose` commands provides diagnostic check support. These are the main focus of the application. Use the following syntax:

```sh
cloudzero-agent-validator diagnose [sub-command] [command options]
```

#### Sub-commands

##### `get-available`

The `get-available` command lists the available diagnostic checks. Use the following syntax:

```sh
cloudzero-agent-validator diagnose get-available
```

##### `run`

The `run-only` command runs a specific check or checks. Use the following syntax:

```sh
cloudzero-agent-validator diagnose run-only [command options]
```

##### `pre-start`

The `pre-start` command runs pre-start diagnostic tests. This command is designed to be used in a `initContainer` pod context of the Cloudzero-Agent chart. Use the following syntax:

```sh
cloudzero-agent-validator diagnose pre-start [command options]
```

##### `post-start`

The `post-start` command runs post-start diagnostic tests. This command is designed to be used in a `lifecycle.PostStart` pod lifecycle context of the Cloudzero-Agent chart. Use the following syntax:

```sh
cloudzero-agent-validator diagnose post-start [command options]
```

##### `pre-stop`

The `pre-stop` command runs pre-stop diagnostic tests.  This command is designed to be used in a `lifecycle.PreStop` pod lifecycle context of the Cloudzero-Agent chart. Use the following syntax:

```sh
cloudzero-agent-validator diagnose pre-stop [command options]
```

--- 

## Global Options
The Cloudzero Agent Validator supports the following global options:

- `--help, -h`: Show help
- `--version, -v`: Print the version

## Copyright
© 2024 Cloudzero, Inc.

