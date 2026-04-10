// ローカル開発用 MinIO プロキシサーバー
// アクセスの都度 MinIO (localhost:9000) の jishinranking-html バケットを参照する。
// 使い方:
//
//	go run ./cmd/serve
//
// http://localhost:8080 でアクセス可能。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	htmlBucketID  = "jishinranking-html"
	minioEndpoint = "http://localhost:9000"
	minioRegion   = "ap-northeast-1"
)

func main() {
	addr := flag.String("addr", ":8080", "listenアドレス (例: :8080)")
	flag.Parse()

	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(minioRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		log.Fatalf("AWS SDK config 初期化失敗: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(minioEndpoint)
		o.UsePathStyle = true
	})

	fmt.Printf("🌐 MinIO プロキシ: %s/%s\n", minioEndpoint, htmlBucketID)
	fmt.Printf("   http://localhost%s\n\n", *addr)

	// /pref_flag/ は static/ ディレクトリからローカル配信
	http.Handle("/pref_flag/", http.StripPrefix("/pref_flag/", http.FileServer(http.Dir("../static/pref_flag"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// trailing slash (except root "/") → 301 redirect to explicit index.html
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"index.html", http.StatusMovedPermanently)
			return
		}
		key := urlToKey(r.URL.Path)
		log.Printf("%s %s → s3://%s/%s", r.Method, r.URL.Path, htmlBucketID, key)

		obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(htmlBucketID),
			Key:    aws.String(key),
		})
		if err != nil {
			var nsk *types.NoSuchKey
			if errors.As(err, &nsk) {
				http.NotFound(w, r)
				return
			}
			log.Printf("S3 GetObject error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer obj.Body.Close()

		w.Header().Set("Content-Type", contentTypeFor(key))
		io.Copy(w, obj.Body)
	})

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

// urlToKey は URL パスを S3 オブジェクトキーに変換する。
//
//	"/"           → "index.html"
//	"/eq/xxx/"    → "eq/xxx/index.html"
//	"/about.html" → "about.html"
//	"/img.png"    → "img.png"
func urlToKey(urlPath string) string {
	p := path.Clean(urlPath)
	if p == "/" || p == "." {
		return "index.html"
	}
	key := strings.TrimPrefix(p, "/")
	// 元のパスが "/" 終わり、または拡張子なし → index.html を補完
	if strings.HasSuffix(urlPath, "/") || path.Ext(key) == "" {
		return key + "/index.html"
	}
	return key
}

// contentTypeFor はキーの拡張子から Content-Type を返す。
func contentTypeFor(key string) string {
	switch path.Ext(key) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".geojson":
		return "application/geo+json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	}
	if ct := mime.TypeByExtension(path.Ext(key)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
