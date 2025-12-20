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
