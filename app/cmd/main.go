package main

import (
	"context"
	"log/slog"
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
	// ==============================
	// 環境変数
	// ==============================
	dataBucketID := os.Getenv("DATA_BUCKET")
	htmlBucketID := os.Getenv("HTML_BUCKET")
	templatePath := os.Getenv("TEMPLATE_PATH")
	// Lambda標準: AWS_REGION が確実。あなたのAWS_REGION_IDはフォールバック扱いに。
	region := getenv("AWS_REGION", getenv("AWS_DEFAULT_REGION", os.Getenv("AWS_REGION_ID")))

	initMode := os.Getenv("INIT") == "true"

	// ==============================
	// 引数取得
	// ==============================

	// ==============================
	// log設定
	// ==============================
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// ==============================
	// context設定
	// ==============================
	// （SDK初期化だけ短めタイムアウト（ハンドラ実行には使わない））
	initCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// ==============================
	// AWS SDK等の初期化
	// ==============================
	awsCfg, err := config.LoadDefaultConfig(
		initCtx,
		config.WithRegion(region),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	s3c := s3.NewFromConfig(awsCfg)

	// ==============================
	// DI
	// ==============================
	dataRepo := datarepo.NewS3DataRepo(s3c, dataBucketID)
	htmlRepo := htmlrepo.NewS3HTMLRepo(s3c, htmlBucketID)
	htmlGen := htmlgen.NewSimpleHTMLGenerator(templatePath)
	h := handler.NewHandler(
		&jma.JMAClient{},
		dataRepo,
		htmlRepo,
		htmlGen,
	)

	// ==============================
	// メイン処理
	// ==============================
	// 1 invoke = 1 Process（これで Runtime.ExitError を潰せる）
	lambda.Start(func(ctx context.Context) error {
		return h.ProcessV2(ctx, initMode)
	})
}
