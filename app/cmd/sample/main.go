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
	// 引数取得
	var (
		init = flag.Bool("init", false, "Initialize the database")
	)
	flag.Parse()

	// Debugレベルで標準出力に出す
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// 環境変数
	dataBucketID := "jishinranking-data"
	htmlBucketID := "jishinranking-html"
	region := "ap-northeast-1"

	ctx := context.Background()

	// datarepo初期化
	datarepoCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	datarepoS3Client := s3.NewFromConfig(datarepoCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
		o.Region = region
	})
	datarepo := datarepo.NewS3DataRepo(datarepoS3Client, dataBucketID)

	// htmlrepo初期化
	htmlrepoCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	s3Client := s3.NewFromConfig(htmlrepoCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
		o.Region = region
	})
	htmlrepo := htmlrepo.NewS3HTMLRepo(s3Client, htmlBucketID)

	// htmlgen初期化
	htmlgen := htmlgen.NewSimpleHTMLGenerator("internal/template")

	// handler初期化
	h := handler.NewHandler(
		&jma.JMAClient{},
		datarepo,
		htmlrepo,
		htmlgen,
	)

	// TODO: これは引数で渡す必要があるか検討
	err = h.Process(ctx, *init)
	if err != nil {
		log.Fatalf("handler.Process error: %v", err)
	}
	log.Println("handler.Process finished successfully")
}
