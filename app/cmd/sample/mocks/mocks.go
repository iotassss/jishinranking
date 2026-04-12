package mocks

import (
	"context"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

type MockJMAFetcher struct{}

func (m *MockJMAFetcher) FetchEarthquakeReport(ctx context.Context) (domain.ReportList, error) {
	return domain.ReportList{}, nil
}

type MockDataRepo struct{}

func (m *MockDataRepo) Save(ctx context.Context, reports domain.ReportList) error {
	return nil
}
func (m *MockDataRepo) Get(ctx context.Context, from, to *time.Time) (domain.ReportList, error) {
	return domain.ReportList{}, nil
}

type MockHTMLRepo struct{}

func (m *MockHTMLRepo) Save(ctx context.Context, key string, html string) error {
	return nil
}
func (m *MockHTMLRepo) SaveRaw(ctx context.Context, key string, content []byte, contentType string) error {
	return nil
}

type MockHTMLGenerator struct{}

func (m *MockHTMLGenerator) Generate(data domain.RankingRecordList) (string, error) {
	return "<html></html>", nil
}
