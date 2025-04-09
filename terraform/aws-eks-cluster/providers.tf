provider "aws" {
  profile = "cz-eng-research"
  region = "us-east-1"
  # Let's be safe and lock down what account this will work with.
  allowed_account_ids = [ "975482786146" ]
  # And this will help ensure we get the tags everywhere we can.
  default_tags {
    tags = var.cluster_tags
  }
}
