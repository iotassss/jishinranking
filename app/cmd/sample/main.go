package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

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

	// ==============================
	// index.htmlをダウンロードして保存（ローカル実行用コード）
	// ==============================
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("project root の検出に失敗: %v", err)
	}

	obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(htmlBucketID),
		Key:    aws.String("index.html"),
	})
	if err != nil {
		log.Fatalf("MinIO から index.html の取得に失敗: %v", err)
	}
	defer obj.Body.Close()

	body, err := io.ReadAll(obj.Body)
	if err != nil {
		log.Fatalf("index.html の読み込みに失敗: %v", err)
	}

	dstPath := filepath.Join(projectRoot, "output", "index.html")
	if err := os.WriteFile(dstPath, body, 0644); err != nil {
		log.Fatalf("index.html の保存に失敗: %v", err)
	}

	log.Printf("MinIO から index.html をダウンロードし、%s に保存しました", dstPath)
}

func findProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, "README.md")); err == nil {
			return cur, nil
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			return "", os.ErrNotExist
		}
		cur = parent
	}
}
