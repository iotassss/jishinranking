.PHONY: build build-app build-init-app

build: build-app build-init-app

build-app:
	cd app && \
	mkdir -p build/linux && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/bootstrap ./cmd/main.go && \
	zip -j build/linux/app.zip build/linux/bootstrap internal/template/index.html
	mkdir -p build
	rm -rf build/linux
	mv app/build/linux/ build/linux/
	rm -rf app/build

build-init-app:
	cd app && \
	mkdir -p build/linux/init_app && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/init_app/bootstrap ./cmd/init/main.go && \
	zip -j build/linux/init_app/initapp.zip build/linux/init_app/bootstrap internal/template/index.html
	mkdir -p build
	rm -rf build/linux/init_app
	mv app/build/linux/init_app/ build/linux/init_app/
	rm -rf app/build
