#!/bin/bash

# This script is run in the Renovate container, before the `renovate` command.
# It is used to install dependencies for the postUpgradeTasks (which are run
# after a dependency is upgraded).
#
# This is only really necessary because we store some generated content in
# version control, and for some dependencies (mostly those related to the Helm
# chart), updating the dependencies can yield slightly different output.
#
# While we run `make install-tools` to install most of our dependencies, there
# are a few tools we just assume are available, that are not available in the
# Renovate image.

apt update && apt install -y \
  golang-go \
  npm \
  patch \
  ${NULL}

# Run the Renovate command.
runuser -u ubuntu renovate
