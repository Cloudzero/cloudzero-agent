# This script merges only the necessary definitions from a JSON schema file into
# the $defs section and updates any $ref paths to use the new $defs section. It
# performs dependency resolution to ensure all transitively referenced
# definitions are included.
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

# Store the k8s definitions for reference
($k8s[0].definitions) as $k8s_defs |

# Function to extract definition name from a $ref path
def extract_def_name($ref):
    if $ref | type == "string" and (($ref | test("^#/\\$defs/")) or ($ref | test("^#/definitions/")))
    then $ref | sub("^#/\\$defs/"; "") | sub("^#/definitions/"; "") | split("/")[0]
    else null
    end;

# Function to find all $ref references in an object
def find_refs:
    [.. | objects | select(has("$ref")) | .["$ref"] | extract_def_name(.) | select(. != null)] | unique;

# Function to recursively collect all dependencies
def collect_dependencies($needed; $k8s_defs):
    $needed as $current_needed |
    (
        # Find all new dependencies from current definitions
        ($current_needed | map(.  as $def_name | $k8s_defs[$def_name] | find_refs) | flatten | unique) as $new_deps |
        
        # Find dependencies that we haven't seen yet
        ($new_deps - $current_needed) as $additional_deps |
        
        if ($additional_deps | length) > 0
        then 
            # Recursively collect dependencies for new definitions
            collect_dependencies($current_needed + $additional_deps; $k8s_defs)
        else 
            $current_needed
        end
    );

# Start with the input and modify it
$input |

# Find all directly referenced definitions in the input
(find_refs) as $direct_refs |

# Collect all needed definitions (including transitive dependencies)
collect_dependencies($direct_refs; $k8s_defs) as $all_needed |

# Build the minimal set of definitions we actually need
($k8s_defs | with_entries(select(.key as $k | $all_needed | contains([$k])))) as $needed_defs |

# Merge only the needed definitions from k8s into the $defs section
.["$defs"] = ($input["$defs"] + $needed_defs) |

# Walk through the entire object and update any $ref paths
#
# The Kubernetes JSON Schema uses the obsolete `definitions` name, but we want
# to put everything in `$defs`.
walk(
    if type == "object" and has("$ref") and (.["$ref"] | type) == "string"
    then .["$ref"] |= sub("^#/definitions/"; "#/$defs/")
    else .
    end
) |

# Remove all description fields to reduce file size (Helm doesn't use them)
walk(
    if type == "object" and has("description")
    then del(.description)
    else .
    end
)
