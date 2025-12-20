package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	ctx := context.Background()

	dataBucket := os.Getenv("DATA_BUCKET")
	htmlBucket := os.Getenv("HTML_BUCKET")
	htmlKey := os.Getenv("HTML_KEY")

	if dataBucket == "" || htmlBucket == "" || htmlKey == "" {
		log.Fatalf("DATA_BUCKET と HTML_BUCKET と HTML_KEY の環境変数を設定してください")
	}

	// AWS SDK 設定 (環境変数や ~/.aws/credentials から取得)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("AWS config の読み込みに失敗しました: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// 出力したいシンプルな HTML
	html := `<!doctype html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>地震ランキング</title>
</head>
<body>
  <h1>地震ランキング</h1>
  <p>Hello, JishinRanking!</p>
</body>
</html>
`

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(htmlBucket),
		Key:         aws.String(htmlKey),
		Body:        strings.NewReader(html),
		ContentType: aws.String("text/html; charset=utf-8"),
	})
	if err != nil {
		log.Fatalf("S3 への HTML アップロードに失敗しました: %v", err)
	}

	log.Printf("HTML を s3://%s/%s に出力しました", htmlBucket, htmlKey)
}
