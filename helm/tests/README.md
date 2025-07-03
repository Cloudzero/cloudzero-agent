# Helm Unit Tests

This directory contains unit tests for the CloudZero Agent Helm chart using the [helm-unittest](https://github.com/helm-unittest/helm-unittest) plugin.

## Overview

The helm unittest plugin allows us to test the actual rendered output of Helm templates, ensuring that:

- Templates render correctly with various input values
- Chart logic works as expected (defaults, overrides, validation)
- Changes to templates don't break expected behavior

## Running Tests

### Prerequisites

The helm unittest plugin must be installed:

```bash
helm plugin install https://github.com/helm-unittest/helm-unittest
```

### Running Tests

```bash
helm unittest .
```
