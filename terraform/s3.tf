# =========================
# Data bucket (private)
# =========================

resource "aws_s3_bucket" "jishinranking_data" {
  bucket = var.data_bucket
}

resource "aws_s3_bucket_public_access_block" "jishinranking_data" {
  bucket = aws_s3_bucket.jishinranking_data.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# =========================
# HTML 配信用 bucket (private に変更)
# =========================
resource "aws_s3_bucket" "jishinranking_html" {
  bucket = var.html_bucket
}

resource "aws_s3_bucket_public_access_block" "jishinranking_html" {
  bucket = aws_s3_bucket.jishinranking_html.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# =========================
# 静的アセット (images)
# =========================

resource "aws_s3_object" "static_favicon" {
  bucket       = aws_s3_bucket.jishinranking_html.id
  key          = "jishinranking_32px.png"
  source       = "../static/jishinranking_32px.png"
  content_type = "image/png"
  source_hash  = filemd5("../static/jishinranking_32px.png")
}

resource "aws_s3_object" "static_icon_256" {
  bucket       = aws_s3_bucket.jishinranking_html.id
  key          = "jishinranking_256px.png"
  source       = "../static/jishinranking_256px.png"
  content_type = "image/png"
  source_hash  = filemd5("../static/jishinranking_256px.png")
}

resource "aws_s3_object" "static_japan_geojson" {
  bucket       = aws_s3_bucket.jishinranking_html.id
  key          = "japan.geojson"
  source       = "../static/japan.geojson"
  content_type = "application/json"
  source_hash  = filemd5("../static/japan.geojson")
}
