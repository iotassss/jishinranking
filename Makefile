.PHONY: build build-app build-init-app serve copy-static

# static/ の静的アセットを output/ にコピーする
copy-static:
	@mkdir -p output
	cp -r static/. output/

# ローカル開発用サーバー: make serve
# MinIO (localhost:9000) をアクセスの都度参照するプロキシとして起動する。
serve:
	cd app && go run ./cmd/serve

build: build-app build-init-app

build-app:
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

build-init-local-app:
	cd app && \
	go run ./cmd/sample/main.go -init
