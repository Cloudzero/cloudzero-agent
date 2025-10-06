# Types Package - Application Core Contracts

## Purpose

Defines interfaces, types, and errors that form the Application Core contracts. All other layers depend on these definitions but this package has no external dependencies.

## Contents

### Core Data Types

- **`Metric`** - Primary metric data structure
- **`ResourceTags`** - Kubernetes resource metadata
- **`AdmissionReview`** - Kubernetes admission webhook types
- **`K8sObject`** - Kubernetes resource abstractions

### Storage Interfaces

- **`WritableStore`** - Metric storage interface
- **`ReadableStore`** - Data retrieval interface
- **`ResourceStore`** - Kubernetes resource metadata storage

### Error Definitions

- **Domain errors** - `ErrNotFound`, `ErrDuplicateKey`, `ErrInvalidData`
- **Validation errors** - `ErrMissingIndices`, `ErrInvalidValue`

### Subdirectories

- **[clusterconfig/](./clusterconfig/)** - Cluster configuration types
- **[status/](./status/)** - Status reporting types
- **[mocks/](./mocks/)** - Generated interface mocks

## Testing

```sh
# Test types package
make test GO_TEST_TARGET=./app/types

# Generate mocks
make generate
```

## Architecture Role

**Application Core** - These types define the contracts between all layers. Changes here affect the entire application.
