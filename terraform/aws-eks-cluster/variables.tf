# see: https://cloudzero.atlassian.net/wiki/spaces/ENG/pages/3817930756/Creating+an+Cluster+using+EKS
variable "csp" {
  type = string
  description = "The cloud service provider the cluster runs in."
  default = "EKS"
  validation {
    condition     = contains(["EKS", "EKS-SCAD", "GKS", "AKS", "OKD" ], var.csp)
    error_message = "Valid values are: (EKS, EKS-SCAD, GKS, AKS, or OKD)."
  }
}

variable "purpose" {
  type = string
  description = "The purpose of the infrastructure."
  default = "test"
  validation {
    condition     = contains(["test", "perf" ], var.purpose) || startswith("CP-", var.purpose)
    error_message = "Valid values are: (test, perf, or CP-XXXX)."
  }
}

variable "team_name" {
  type = string
  description = "Specifies the team responsible for the cluster."
  default = "Cirrus"
}

variable "cluster_context" {
  type = string
  description = "An optional descriptive name for the cluster, providing additional context or details."
  default = null
}

variable "cluster_tags" {
  # see: https://cloudzero.atlassian.net/wiki/spaces/ENG/pages/1505394698/CloudZero+Cloud+Tagging+Policy
  type = map
  description = "A set of tags that should be applied to everything possible."
  # Note: Not all character's are allowed (apostrophes, etc.)
  default = {
    "cz:feature": "Team Cirrus Agent Testing Cluster",
    "cz:owner": "adam.rice@cloudzero.com",
    "cz:team": "cirrus@cloudzero.com",
    "cz:description": "Infrastructure for Team Cirrus Agent Testing",
    "cz:repo": "github.com/cloudzero/cloudzero-agent",
    "cz:namespace": "alfa",
    "cz:env": "research",
    "cz:customer-data": false
  }
}
