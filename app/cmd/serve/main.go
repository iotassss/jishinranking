// ローカル開発用 静的ファイルサーバー
// 使い方:
//
//	go run ./cmd/serve [ディレクトリ]
//
// デフォルトは ../output を公開。 http://localhost:8080 でアクセス可能。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	addr := flag.String("addr", ":8080", "listenアドレス (例: :8080)")
	flag.Parse()

	dir := flag.Arg(0)
	if dir == "" {
		// デフォルト: このファイルから見た ../output
		exe, err := os.Executable()
		if err == nil {
			dir = filepath.Join(filepath.Dir(exe), "../output")
		} else {
			dir = "../output"
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("ディレクトリの解決に失敗しました: %v", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Fatalf("ディレクトリが存在しません: %s", absDir)
	}

	fmt.Printf("🌐 Serving: %s\n", absDir)
	fmt.Printf("   http://localhost%s\n\n", *addr)

	fs := http.FileServer(http.Dir(absDir))
	http.Handle("/", loggingMiddleware(fs))

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
