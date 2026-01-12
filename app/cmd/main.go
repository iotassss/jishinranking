package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/iotassss/jishinranking/internal/datarepo"
	"github.com/iotassss/jishinranking/internal/handler"
	"github.com/iotassss/jishinranking/internal/htmlgen"
	"github.com/iotassss/jishinranking/internal/htmlrepo"
	"github.com/iotassss/jishinranking/internal/jma"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// env
	dataBucketID := os.Getenv("DATA_BUCKET")
	htmlBucketID := os.Getenv("HTML_BUCKET")
	templatePath := os.Getenv("TEMPLATE_PATH")

	// Lambda標準: AWS_REGION が確実。あなたのAWS_REGION_IDはフォールバック扱いに。
	region := getenv("AWS_REGION", getenv("AWS_DEFAULT_REGION", os.Getenv("AWS_REGION_ID")))

	initMode := os.Getenv("INIT") == "true"

	// SDK初期化だけ短めタイムアウト（ハンドラ実行には使わない）
	initCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(initCtx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	s3c := s3.NewFromConfig(cfg)

	dataRepo := datarepo.NewS3DataRepo(s3c, dataBucketID)
	htmlRepo := htmlrepo.NewS3HTMLRepo(s3c, htmlBucketID)
	htmlGen := htmlgen.NewSimpleHTMLGenerator(templatePath)

	h := handler.NewHandler(&jma.JMAClient{}, dataRepo, htmlRepo, htmlGen)

	// 1 invoke = 1 Process（これで Runtime.ExitError を潰せる）
	lambda.Start(func(ctx context.Context) error {
		return h.Process(ctx, initMode)
	})
}
