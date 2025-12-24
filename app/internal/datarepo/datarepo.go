package datarepo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iotassss/jishinranking/internal/domain"
)

type S3DataRepo struct {
	client     *s3.Client
	bucketName string
}

func NewS3DataRepo(client *s3.Client, bucket string) *S3DataRepo {
	return &S3DataRepo{
		client:     client,
		bucketName: bucket,
	}
}

func (r *S3DataRepo) Save(ctx context.Context, reports domain.ReportList) error {
	fileName := domain.NewDataFileNameFromTime(time.Now())
	key := fileName.String()

	data, err := json.Marshal(reports)
	if err != nil {
		return fmt.Errorf("failed to marshal reports to JSON: %w", err)
	}

	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("S3 PutObject failed: %w", err)
	}

	return nil
}

func (r *S3DataRepo) Get(ctx context.Context, from, to *time.Time) (domain.ReportList, error) {
	if from != nil && to != nil && from.After(*to) {
		return nil, fmt.Errorf("invalid time range: from is after to")
	}

	// ファイル名の一覧を取得
	resp, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 ListObjects failed: %w", err)
	}

	// from, toでフィルタリング
	// ファイル名はyyyyMMddThhmmssZ.json形式なので、文字列比較でフィルタリング可能
	// 各ファイルを取得してパースし、ReportListにまとめて返す
	var keys []string
	for _, obj := range resp.Contents {
		key := *obj.Key
		fileName := domain.DataFileName(key)
		fileTime, err := fileName.ToTime()
		if err != nil {
			return nil, fmt.Errorf("invalid data file name %s: %w", key, err)
		}

		if from != nil && fileTime.Before(*from) {
			continue
		}
		if to != nil && fileTime.After(*to) {
			continue
		}

		keys = append(keys, key)
	}

	// 取得したファイル名一覧からデータを取得してパース
	var allReports domain.ReportList
	for _, key := range keys {
		resp, err := r.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(r.bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, fmt.Errorf("S3 GetObject failed: %w", err)
		}
		defer resp.Body.Close()

		var reports domain.ReportList
		if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
			return nil, fmt.Errorf("failed to decode JSON from %s: %w", key, err)
		}
		allReports = append(allReports, reports...)
	}

	return allReports, nil
}
