# This script merges the definitions from a JSON schema file (only tested with
# https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/master/_definitions.json
# which has been copied into this repository) into the $defs section and updates
# any $ref paths to use the new $defs section.
#
# Usage:
#
#   gojq --yaml-input . helm/values.schema.yaml | \
#     gojq --slurpfile k8s helm/schema/k8s.json -f scripts/merge-json-schema.jq > helm/values.schema.json
#
# Or, better yet, just use the Makefile target:
#
#   make helm/values.schema.json

# Store the input in a variable for later use
. as $input |

# Start with the input and modify it
$input |

# Merge the definitions from k8s into the $defs section
.["$defs"] = ($input["$defs"] + ($k8s[0].definitions)) |

# Walk through the entire object and update any $ref paths
#
# The Kubernetes JSON Schema uses the obsolete `definitions` name, but we want
# to put everything in `$defs`.
walk(
    if type == "object" and has("$ref") and (.["$ref"] | type) == "string"
    then .["$ref"] |= sub("^#/definitions/"; "#/$defs/")
    else .
    end
)
