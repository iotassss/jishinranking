package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iotassss/jishinranking/internal/datarepo"
	"github.com/iotassss/jishinranking/internal/handler"
	"github.com/iotassss/jishinranking/internal/htmlgen"
	"github.com/iotassss/jishinranking/internal/htmlrepo"
	"github.com/iotassss/jishinranking/internal/jma"
)

func main() {
	// 環境変数
	dataBucketID := os.Getenv("DATA_BUCKET")
	htmlBucketID := os.Getenv("HTML_BUCKET")
	region := os.Getenv("AWS_REGION_ID")
	templatePath := os.Getenv("TEMPLATE_PATH")
	var init bool
	if os.Getenv("INIT") == "true" {
		init = true
	} else if os.Getenv("INIT") == "false" || os.Getenv("INIT") == "" {
		init = false
	} else {
		panic("ENV INIT must be 'true' or 'false'")
	}

	ctx := context.Background()

	// datarepo初期化
	datarepoCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	datarepoS3Client := s3.NewFromConfig(datarepoCfg)
	datarepo := datarepo.NewS3DataRepo(datarepoS3Client, dataBucketID)

	// htmlrepo初期化
	htmlrepoCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	s3Client := s3.NewFromConfig(htmlrepoCfg)
	htmlrepo := htmlrepo.NewS3HTMLRepo(s3Client, htmlBucketID)

	// htmlgen初期化
	htmlgen := htmlgen.NewSimpleHTMLGenerator(templatePath)

	// handler初期化
	h := handler.NewHandler(
		&jma.JMAClient{},
		datarepo,
		htmlrepo,
		htmlgen,
	)

	err = h.Process(ctx, init)
	if err != nil {
		log.Fatalf("handler.Process error: %v", err)
	}
	log.Println("handler.Process finished successfully")
}
