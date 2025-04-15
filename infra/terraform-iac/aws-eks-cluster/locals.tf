locals {
  name   = "${var.csp}-${var.purpose}-${var.team_name}-${var.cluster_context}"
  region = "us-east-1"

  vpc_cidr = "10.0.0.0/16"
  azs      = slice(data.aws_availability_zones.available.names, 0, 3)

  tags = merge(local.default_tags, var.cluster_tags)

  default_tags = {
    automated = "true"
    codebase  = "github.com/cloudzero/cloudzero-agent/terraform/aws-eks-cluster"
  }
}
