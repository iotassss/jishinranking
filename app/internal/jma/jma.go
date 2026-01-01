// internal/jma/jma.go
package jma

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/iotassss/jishinranking/internal/domain"
)

// const eqvolFeedURL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol.xml"

// 長期観測用フィード
const eqvolFeedURL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol_l.xml"

type JMAClient struct{}

func (c *JMAClient) FetchEQVOLFeed(ctx context.Context) (domain.Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eqvolFeedURL, nil)
	if err != nil {
		return domain.Feed{}, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Feed{}, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return domain.Feed{}, fmt.Errorf("http %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.Feed{}, fmt.Errorf("read body: %w", err)
	}

	var feed domain.Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return domain.Feed{}, fmt.Errorf("xml unmarshal feed: %w", err)
	}

	return feed, nil
}

func (c *JMAClient) FetchEarthquakeReport(ctx context.Context, urls []string) (domain.ReportList, error) {
	var reports []domain.Report
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http get: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return nil, fmt.Errorf("http %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		var report domain.Report
		if err := xml.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("xml unmarshal: %w", err)
		}
		report.Metadata.URL = url

		reports = append(reports, report)
	}

	return reports, nil
}
