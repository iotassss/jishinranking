package htmlgen

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

// 地震イベント構造体（最低限）
type EarthquakeEvent struct {
	Pref string `json:"pref"`
}

type SimpleHTMLGenerator struct {
	tmpl *template.Template
}

func NewSimpleHTMLGenerator(templatePath string) *SimpleHTMLGenerator {
	// テンプレートファイルのパス
	tmplPath := filepath.Join(templatePath, "index.html")
	// 関数マップ（順位表示用）
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		// 開発時はpanic、運用時はエラー返却推奨
		panic(fmt.Sprintf("template parse error: %v", err))
	}
	return &SimpleHTMLGenerator{tmpl: tmpl}
}

// TODO: 回数のランキングと最大震度のランキングを含むHTMLを生成できるように変更する
// Generate: 地震イベントJSONからHTMLランキング表を生成
// func (g *SimpleHTMLGenerator) Generate(data []byte) ([]byte, error) {
func (g *SimpleHTMLGenerator) Generate(
	displayData domain.DisplayData,
	from, to, now time.Time,
) (domain.PublishedHTML, error) {
	// テンプレート描画
	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"Ranking":             displayData.RankingRecords,
		"LatestEarthquakes":   displayData.LatestEarthquakes,
		"TodayBigEarthquakes": displayData.TodayBigEarthquakes,
		"WeekBigEarthquakes":  displayData.WeekBigEarthquakes,
		"MonthBigEarthquakes": displayData.MonthBigEarthquakes,
		"ReportPeriod":        from.Format("2006-01-02") + " ～ " + to.Format("2006-01-02"),
		"Now":                 now.Format(time.RFC3339),
	}
	if err := g.tmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}
