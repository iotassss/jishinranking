package htmlgen

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"

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
	tmplPath := filepath.Join(templatePath, "index.tmpl")
	// 関数マップ（順位表示用）
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
	}
	tmpl, err := template.New("index.tmpl").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		// 開発時はpanic、運用時はエラー返却推奨
		panic(fmt.Sprintf("template parse error: %v", err))
	}
	return &SimpleHTMLGenerator{tmpl: tmpl}
}

// TODO: 回数のランキングと最大震度のランキングを含むHTMLを生成できるように変更する
// Generate: 地震イベントJSONからHTMLランキング表を生成
// func (g *SimpleHTMLGenerator) Generate(data []byte) ([]byte, error) {
func (g *SimpleHTMLGenerator) Generate(records domain.RankingRecordList) (string, error) {

	// recordsを降順ソート（Count）
	records.SortByCountDesc()
	// テンプレート描画
	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"Ranking": records,
	}
	if err := g.tmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("template execute failed: %w", err)
	}
	return buf.String(), nil
}
