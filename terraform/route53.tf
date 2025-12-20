###############################################################################
# Route53 zone (既存の Hosted Zone を参照)
# まだ作ってないなら data ではなく aws_route53_zone を作る
###############################################################################

resource "aws_route53_zone" "this" {
  name         = var.domain
}

output "route53_nameservers" {
  value = aws_route53_zone.this.name_servers
}

###############################################################################
# ACM DNS validation records (Route53)
###############################################################################

resource "aws_route53_record" "acm_validation" {
  for_each = {
    for dvo in aws_acm_certificate.cf.domain_validation_options :
    dvo.domain_name => {
      name  = dvo.resource_record_name
      type  = dvo.resource_record_type
      value = dvo.resource_record_value
    }
  }

  zone_id = aws_route53_zone.this.zone_id
  name    = each.value.name
  type    = each.value.type
  records = [each.value.value]
  ttl     = 60
}

resource "aws_acm_certificate_validation" "cf" {
  provider                = aws.use1
  certificate_arn         = aws_acm_certificate.cf.arn
  validation_record_fqdns = [for r in aws_route53_record.acm_validation : r.fqdn]
}

###############################################################################
# Route53 -> CloudFront (apex)
###############################################################################

resource "aws_route53_record" "apex_a" {
  zone_id = aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.this.domain_name
    zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "apex_aaaa" {
  zone_id = aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.this.domain_name
    zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
    evaluate_target_health = false
  }
}

# www も使うなら（cloudfront.tf の aliases / ACM SAN も合わせてON）
# resource "aws_route53_record" "www_a" {
#   zone_id = data.aws_route53_zone.this.zone_id
#   name    = "www.${var.domain}"
#   type    = "A"
#   alias {
#     name                   = aws_cloudfront_distribution.this.domain_name
#     zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
#     evaluate_target_health = false
#   }
# }
# resource "aws_route53_record" "www_aaaa" {
#   zone_id = data.aws_route53_zone.this.zone_id
#   name    = "www.${var.domain}"
#   type    = "AAAA"
#   alias {
#     name                   = aws_cloudfront_distribution.this.domain_name
#     zone_id                = aws_cloudfront_distribution.this.hosted_zone_id
#     evaluate_target_health = false
#   }
# }
