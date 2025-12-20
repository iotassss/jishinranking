# variables.tf
###############################################################################
# shared
###############################################################################
variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "ap-northeast-1"
}

###############################################################################
# Cloudflare identifiers
###############################################################################
variable "cloudflare_account_id" {
  type        = string
  description = "Cloudflare Account ID"
}

variable "cloudflare_zone_id" {
  type        = string
  description = "Cloudflare Zone ID"
}

variable "domain" {
  type        = string
  description = "Zone apex domain (e.g. example.com)"
  default     = "oninnoran.com"
}

###############################################################################
# S3 origin (private)
###############################################################################
variable "data_bucket" {
  type        = string
  description = "Private S3 bucket name that stores HTML"
  default     = "jishinranking-data"
}

variable "html_bucket" {
  type        = string
  description = "Private S3 bucket name that stores HTML"
  default     = "jishinranking-html"
}

variable "s3_prefix" {
  type        = string
  description = "Prefix within S3 bucket (e.g. public/html/). Empty means bucket root."
  default     = ""
}

###############################################################################
# AWS creds for Worker (SigV4)
# NOTE: これを TF_VAR_* で渡すと state に残り得るので運用注意
###############################################################################
variable "worker_aws_access_key_id" {
  type        = string
  description = "AWS access key id used by Cloudflare Worker to sign S3 requests"
  sensitive   = true
}

variable "worker_aws_secret_access_key" {
  type        = string
  description = "AWS secret access key used by Cloudflare Worker to sign S3 requests"
  sensitive   = true
}
