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

const eqvolFeedURL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol.xml"

// 長期観測用フィード
// const eqvolFeedURL = "https://www.data.jma.go.jp/developer/xml/feed/eqvol_l.xml"

var earthquakeFilterWords = []string{
	"地震", "震度", "揺れ", "マグニチュード",
}

type JMAClient struct{}

func (c *JMAClient) FetchEarthquakeReport(ctx context.Context) (domain.ReportList, error) {
	// Feedにアクセスしてxmlデータを取得
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eqvolFeedURL, nil)
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

	var feed domain.Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("xml unmarshal feed: %w", err)
	}

	earthquakeEntries := filterEarthquakeEntries(feed)
	if len(earthquakeEntries) == 0 {
		return nil, nil
	}

	// 地震エントリーのリストからそれぞれの地震データを取得パースする

	var reports []domain.Report
	for _, entry := range earthquakeEntries {
		// 3. それぞれの地震エントリーのurlにアクセスしてXMLを取得・パースする
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.Link.Href, nil)
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
		report.Metadata.URL = entry.Link.Href

		reports = append(reports, report)
	}

	return reports, nil
}

// earthquakeFilterWordsのワードがentry.titleに含まれていれば地震エントリーとみなす
func filterEarthquakeEntries(feed domain.Feed) []domain.Entry {
	var earthquakeEntries []domain.Entry
	for _, entry := range feed.Entries {
		for _, word := range earthquakeFilterWords {
			if strings.Contains(entry.Title, word) || strings.Contains(entry.Content.Text, word) {
				earthquakeEntries = append(earthquakeEntries, entry)
				break
			}
		}
	}
	return earthquakeEntries
}
