# Helm Template Generation Tests

These are used to make sure all modifications to the Helm chart are intentional,
and have the desired effect (and no more).

To add an additional test, you'll want two files: a `foo.yaml` and
`foo-overrides.yml`. Note the differing extensions; we use that difference in
the makefile in order to distinguish between the differing functionality.

To (re)generate a `foo.yaml`, first make sure your `foo-overrides.yml` is in
place, then run `make tests/helm/template/foo.yaml`.
