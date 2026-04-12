.PHONY: build build-app build-init-app serve copy-static gentestdata makepages makepagesinit resetlocal

# static/ の静的アセットを output/ にコピーする
copy-static:
	@mkdir -p output
	cp -r static/. output/

# =====================
# 本番アプリケーションビルド
# =====================

# 本番アプリケーションビルド
# terraformでLambdaにアップロードするzipファイルを生成する
build:
	cd app && \
	mkdir -p build/linux && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/bootstrap ./cmd/main.go && \
	zip -j build/linux/app.zip build/linux/bootstrap && \
	cd internal && zip -r ../build/linux/app.zip template/ --exclude '*.bak'
	mkdir -p build
	rm -rf build/linux
	mv app/build/linux/ build/linux/
	rm -rf app/build
	rm build/linux/bootstrap

# =====================
# ローカル開発用アプリケーションビルド
# =====================

# テストデータ生成: json生成 → MinIO へアップロード
# 事前に mc alias set local http://localhost:9000 minioadmin minioadmin が必要
gentestdata:
	rm -rf tmp/testdata
	mkdir -p tmp/testdata
	cd app && go run ./cmd/gendata/ > ../tmp/testdata/$$(date -u +%Y%m%dT%H%M%SZ).json
	mc rm --recursive --force local/jishinranking-data
	mc cp "$$(printf "%s\n" /Users/iota/Workspace/jishinranking/tmp/testdata/[0-9]*T*.json | sort | tail -n 1)" local/jishinranking-data/

makepages:
	cd app && \
	go run ./cmd/sample/main.go

makepagesinit:
	cd app && \
	go run ./cmd/sample/main.go -init

resetlocal: gentestdata makepages

# ローカル開発用サーバー: make serve
# MinIO (localhost:9000) をアクセスの都度参照するプロキシとして起動する。
serve:
	$(MAKE) resetlocal
	cd app && go run ./cmd/serve
