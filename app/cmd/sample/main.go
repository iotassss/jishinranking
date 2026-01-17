package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iotassss/jishinranking/internal/datarepo"
	"github.com/iotassss/jishinranking/internal/handler"
	"github.com/iotassss/jishinranking/internal/htmlgen"
	"github.com/iotassss/jishinranking/internal/htmlrepo"
	"github.com/iotassss/jishinranking/internal/jma"
)

func main() {
	// ==============================
	// 環境変数
	// ==============================
	dataBucketID := "jishinranking-data"
	htmlBucketID := "jishinranking-html"
	region := "ap-northeast-1"

	// ==============================
	// 引数取得
	// ==============================
	var (
		init = flag.Bool("init", false, "Initialize the database")
	)
	flag.Parse()

	// ==============================
	// log設定
	// ==============================
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// ==============================
	// context設定
	// ==============================
	ctx := context.Background()

	// ==============================
	// AWS SDK等の初期
	// ==============================
	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
		o.Region = "ap-northeast-1"
	})

	// ==============================
	// DI
	// ==============================
	datarepo := datarepo.NewS3DataRepo(s3Client, dataBucketID)
	htmlrepo := htmlrepo.NewS3HTMLRepo(s3Client, htmlBucketID)
	htmlgen := htmlgen.NewSimpleHTMLGenerator("internal/template")
	h := handler.NewHandler(
		&jma.JMAClient{},
		datarepo,
		htmlrepo,
		htmlgen,
	)

	// ==============================
	// メイン処理
	// ==============================
	// TODO: これは引数で渡す必要があるか検討
	err = h.Process(ctx, *init)
	if err != nil {
		log.Fatalf("handler.Process error: %v", err)
	}
	log.Println("handler.Process finished successfully")
}
