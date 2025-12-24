package htmlrepo

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3HTMLRepo struct {
	Client     *s3.Client
	BucketName string
}

func NewS3HTMLRepo(client *s3.Client, bucket string) *S3HTMLRepo {
	return &S3HTMLRepo{
		Client:     client,
		BucketName: bucket,
	}
}

// SaveはkeyにHTMLデータを書き込む
func (r *S3HTMLRepo) Save(ctx context.Context, key string, html string) error {
	_, err := r.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(html)),
		ContentType: aws.String("text/html; charset=utf-8"),
	})
	if err != nil {
		return fmt.Errorf("S3 PutObject failed: %w", err)
	}
	return nil
}
