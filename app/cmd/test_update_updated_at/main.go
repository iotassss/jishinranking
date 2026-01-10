package main

import (
	"fmt"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

func main() {
	// テスト用のHTML（<time>タグと集計期間両方を含む）
	html := `<html><body><time id="updated-at" datetime="2025-12-31T23:59:59+09:00" style="display:none">2025-12-31T23:59:59+09:00</time><div>集計期間: 2025-11-27 ～ 2026-12-03</div></body></html>`
	p := domain.PublishedHTML(html)
	// JSTでテスト（+09:00）
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Date(2026, 1, 3, 15, 54, 32, 0, jst)
	// UpdateUpdatedAtのテスト
	updatedTime := p.UpdateUpdatedAt(now)
	fmt.Println("--- Before ---")
	fmt.Println(html)
	fmt.Println("--- After UpdateUpdatedAt ---")
	fmt.Println(string(updatedTime))

	// UpdatePeriodのテスト
	from := time.Date(2025, 12, 27, 0, 0, 0, 0, jst)
	to := time.Date(2026, 1, 3, 0, 0, 0, 0, jst)
	updatedPeriod := p.UpdatePeriod(from, to)
	fmt.Println("--- After UpdatePeriod ---")
	fmt.Println(string(updatedPeriod))

	// 期待される置換結果例
	expectedTime := now.Format(time.RFC3339)
	expectedPeriod := from.Format("2006-01-02") + " ～ " + to.Format("2006-01-02")

	fmt.Println("--- Check ---")
	if !contains(string(updatedTime), expectedTime) {
		fmt.Println("NG: <time>タグの置換に失敗")
	} else {
		fmt.Println("OK: <time>タグの置換成功")
	}
	if !contains(string(updatedPeriod), expectedPeriod) {
		fmt.Println("NG: 集計期間の置換に失敗")
	} else {
		fmt.Println("OK: 集計期間の置換成功")
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
