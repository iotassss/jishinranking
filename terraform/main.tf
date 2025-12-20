terraform {
  required_version = ">= 1.8.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket  = "jishinranking-tfstate"
    key     = "terraform/state.tfstate"
    region  = "ap-northeast-1"
    encrypt = true
  }
}

###############################################################################
# Providers
###############################################################################
provider "aws" {
  region = var.aws_region
}

# us-east-1 (CloudFront ACM用)
provider "aws" {
  alias  = "use1"
  region = "us-east-1"
}
