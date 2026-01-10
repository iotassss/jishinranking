resource "aws_lambda_function" "batch" {
  function_name = "jishinranking-batch"
  role          = aws_iam_role.lambda.arn

  package_type = "Zip"
  runtime      = "provided.al2023"
  handler      = "bootstrap"

  filename         = "../build/linux/app.zip"
  source_code_hash = filebase64sha256("../build/linux/app.zip")

  memory_size = 512
  timeout     = 15

  environment {
    variables = {
      DATA_BUCKET  = aws_s3_bucket.jishinranking_data.bucket
      HTML_BUCKET  = aws_s3_bucket.jishinranking_html.bucket
      JSON_KEY     = "data/events.json"
      HTML_KEY     = "index.html"
      JMA_FEED_URL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol.xml"
      DAYS_RANGE   = "7"
      AWS_REGION_ID   = var.aws_region
      TEMPLATE_PATH = ""
    }
  }
}


resource "aws_lambda_function" "initialize" {
  function_name = "jishinranking-initialize"
  role          = aws_iam_role.lambda.arn

  package_type = "Zip"
  runtime      = "provided.al2023"
  handler      = "bootstrap"

  filename         = "../build/linux/init_app/initapp.zip"
  source_code_hash = filebase64sha256("../build/linux/init_app/initapp.zip")

  memory_size = 512
  timeout     = 15

  environment {
    variables = {
      DATA_BUCKET  = aws_s3_bucket.jishinranking_data.bucket
      HTML_BUCKET  = aws_s3_bucket.jishinranking_html.bucket
      JSON_KEY     = "data/events.json"
      HTML_KEY     = "index.html"
      JMA_FEED_URL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol_l.xml"
      DAYS_RANGE   = "7"
      AWS_REGION_ID   = var.aws_region
      TEMPLATE_PATH = ""
    }
  }
}
