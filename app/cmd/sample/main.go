package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

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

const (
	dataBucketID = "jishinranking-data"
	htmlBucketID = "jishinranking-html"
)

func main() {
	// Debugレベルで標準出力に出す
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// 環境変数

	ctx := context.Background()

	// datarepo初期化
	datarepoCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	datarepoS3Client := s3.NewFromConfig(datarepoCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
		o.Region = "ap-northeast-1"
	})
	datarepo := datarepo.NewS3DataRepo(datarepoS3Client, dataBucketID)

	// htmlrepo初期化
	htmlrepoCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	s3Client := s3.NewFromConfig(htmlrepoCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
		o.Region = "ap-northeast-1"
	})
	htmlrepo := htmlrepo.NewS3HTMLRepo(s3Client, htmlBucketID)

	// htmlgen初期化
	htmlgen := htmlgen.NewSimpleHTMLGenerator()

	// handler初期化
	h := handler.NewHandler(
		&jma.JMAClient{},
		datarepo,
		htmlrepo,
		htmlgen,
	)

	dataKey := time.Now().UTC().Format("20060102T150405Z.json")
	htmlKey := "index.html"

	// TODO: これは引数で渡す必要があるか検討
	err = h.Process(ctx, dataKey, htmlKey)
	if err != nil {
		log.Fatalf("handler.Process error: %v", err)
	}
	log.Println("handler.Process finished successfully")
}
