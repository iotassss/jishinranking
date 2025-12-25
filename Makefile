.PHONY: build
build:
	cd app && \
	mkdir -p build/linux && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/bootstrap ./cmd/main.go && \
	zip -j build/linux/app.zip build/linux/bootstrap internal/template/index.tmpl
	mkdir -p build
	rm -rf build/linux
	mv app/build/linux/ build/linux/
	rm -rf app/build
