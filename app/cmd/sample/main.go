package main

import (
	"bytes"
	"context"
	"flag"
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
	err = h.ProcessV2(ctx, *init)
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

	// ==============================
	// static/ の静的アセットを MinIO にアップロード
	// ==============================
	staticDir := filepath.Join(projectRoot, "static")
	entries, err := os.ReadDir(staticDir)
	if err != nil {
		log.Printf("static/ ディレクトリの読み込みに失敗（スキップ）: %v", err)
	} else {
		uploadCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			data, err := os.ReadFile(filepath.Join(staticDir, name))
			if err != nil {
				log.Printf("static/%s の読み込みに失敗: %v", name, err)
				continue
			}
			ct := staticContentType(name)
			_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(htmlBucketID),
				Key:         aws.String(name),
				Body:        bytes.NewReader(data),
				ContentType: aws.String(ct),
			})
			if err != nil {
				log.Printf("static/%s のアップロードに失敗: %v", name, err)
				continue
			}
			log.Printf("static/%s → MinIO にアップロード完了 (%s)", name, ct)
			uploadCount++
		}
		log.Printf("静的アセットを %d 件アップロードしました", uploadCount)
	}
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

// staticContentType はファイル名の拡張子から Content-Type を返す。
func staticContentType(name string) string {
	switch filepath.Ext(name) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	}
	return "application/octet-stream"
}
