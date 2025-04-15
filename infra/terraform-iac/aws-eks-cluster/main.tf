terraform {
  required_version = ">= 1.11.4"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.94"
    }
  }
  # We will likely want bucket logging and tight permissions added to this bucket.
  backend "s3" {
    bucket       = "cz-eng-research-team-cirrus-terraform-state"
    key          = "tfstate/cloudzero-agent/terraform/aws-eks-cluster/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = "true"
  }
}
