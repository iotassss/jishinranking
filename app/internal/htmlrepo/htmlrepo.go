package htmlrepo

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iotassss/jishinranking/internal/domain"
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
func (r *S3HTMLRepo) Save(ctx context.Context, key string, html domain.PublishedHTML) error {
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

// SaveRaw はkeyに任意のバイト列をcontentTypeとして書き込む
func (r *S3HTMLRepo) SaveRaw(ctx context.Context, key string, content []byte, contentType string) error {
	_, err := r.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("S3 PutObject failed: %w", err)
	}
	return nil
}

func (r *S3HTMLRepo) Get(ctx context.Context, key string) (domain.PublishedHTML, error) {
	resp, err := r.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("S3 GetObject failed: %w", err)
	}
	defer resp.Body.Close()

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return domain.PublishedHTML(html), nil
}
