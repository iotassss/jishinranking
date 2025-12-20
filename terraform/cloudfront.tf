###############################################################################
# ACM (us-east-1) for CloudFront
###############################################################################

resource "aws_acm_certificate" "cf" {
  provider          = aws.use1
  domain_name       = var.domain
  validation_method = "DNS"

  # www も使うなら SAN を追加し、cloudfront aliases / route53 も増やす
  # subject_alternative_names = ["www.${var.domain}"]

  lifecycle {
    create_before_destroy = true
  }
}

###############################################################################
# CloudFront OAC
###############################################################################

resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = "${var.domain}-oac"
  description                       = "OAC for private S3 origin"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

###############################################################################
# CloudFront distribution
###############################################################################

resource "aws_cloudfront_distribution" "this" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = var.domain
  default_root_object = "index.html"

  aliases = [var.domain]
  # www も使うなら:
  # aliases = [var.domain, "www.${var.domain}"]

  origin {
    domain_name              = aws_s3_bucket.jishinranking_html.bucket_regional_domain_name
    origin_id                = "s3-${aws_s3_bucket.jishinranking_html.id}"
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
  }

  default_cache_behavior {
    target_origin_id       = "s3-${aws_s3_bucket.jishinranking_html.id}"
    viewer_protocol_policy = "redirect-to-https"

    allowed_methods = ["GET", "HEAD"]
    cached_methods  = ["GET", "HEAD"]

    compress = true

    forwarded_values {
      query_string = false
      cookies { forward = "none" }
    }
  }

  # SPAなら 404/403 を index.html に寄せる（必要なら有効化）
  # custom_error_response {
  #   error_code            = 403
  #   response_code         = 200
  #   response_page_path    = "/index.html"
  #   error_caching_min_ttl = 0
  # }
  # custom_error_response {
  #   error_code            = 404
  #   response_code         = 200
  #   response_page_path    = "/index.html"
  #   error_caching_min_ttl = 0
  # }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    # DNS 検証完了後の ARN（route53.tf の aws_acm_certificate_validation.cf）
    acm_certificate_arn      = aws_acm_certificate_validation.cf.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  # コスト寄せたいなら
  # price_class = "PriceClass_200"
}

###############################################################################
# S3 bucket policy: CloudFront(OAC) からの GetObject のみ許可
###############################################################################

data "aws_iam_policy_document" "allow_cloudfront_oac" {
  statement {
    sid     = "AllowCloudFrontServicePrincipalReadOnly"
    effect  = "Allow"
    actions = ["s3:GetObject"]

    resources = ["${aws_s3_bucket.jishinranking_html.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.this.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "jishinranking_html" {
  bucket = aws_s3_bucket.jishinranking_html.id
  policy = data.aws_iam_policy_document.allow_cloudfront_oac.json
}

output "cloudfront_domain_name" {
  value = aws_cloudfront_distribution.this.domain_name
}
