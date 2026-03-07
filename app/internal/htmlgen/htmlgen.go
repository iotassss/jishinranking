package htmlgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
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

// ---- hourly chart helpers ----

type hourlyChartDataJSON struct {
	Labels   []string            `json:"labels"`
	Datasets []hourlyDatasetJSON `json:"datasets"`
}

type hourlyDatasetJSON struct {
	Label           string `json:"label"`
	Data            []int  `json:"data"`
	BackgroundColor string `json:"backgroundColor"`
	Stack           string `json:"stack"`
}

// hourlyChartColors は都道府県コード順（1=北海道〜47=沖縄）に対応した固定色パレット。
var hourlyChartColors = []string{
	"#e6194b", // 1  北海道
	"#3cb44b", // 2  青森
	"#ffe119", // 3  岩手
	"#4363d8", // 4  宮城
	"#f58231", // 5  秋田
	"#911eb4", // 6  山形
	"#42d4f4", // 7  福島
	"#f032e6", // 8  茨城
	"#bfef45", // 9  栃木
	"#fabed4", // 10 群馬
	"#469990", // 11 埼玉
	"#dcbeff", // 12 千葉
	"#9a6324", // 13 東京
	"#fffac8", // 14 神奈川
	"#800000", // 15 新潟
	"#aaffc3", // 16 富山
	"#808000", // 17 石川
	"#ffd8b1", // 18 福井
	"#000075", // 19 山梨
	"#a9a9a9", // 20 長野
	"#ff4040", // 21 岐阜
	"#008080", // 22 静岡
	"#4040ff", // 23 愛知
	"#ff8040", // 24 三重
	"#40c040", // 25 滋賀
	"#c040c0", // 26 京都
	"#40c0c0", // 27 大阪
	"#c0c040", // 28 兵庫
	"#804040", // 29 奈良
	"#408080", // 30 和歌山
	"#804080", // 31 鳥取
	"#808040", // 32 島根
	"#4080c0", // 33 岡山
	"#c08040", // 34 広島
	"#40c080", // 35 山口
	"#c04080", // 36 徳島
	"#8040c0", // 37 香川
	"#80c040", // 38 愛媛
	"#c04040", // 39 高知
	"#4040c0", // 40 福岡
	"#40c0c0", // 41 佐賀
	"#c0c040", // 42 長崎
	"#ff6080", // 43 熊本
	"#60c060", // 44 大分
	"#6060ff", // 45 宮崎
	"#ff60a0", // 46 鹿児島
	"#00bfff", // 47 沖縄
}

// makeHourlyChartData は HourlyEarthquakeData を Chart.js の積み上げ棒グラフ用 JSON に変換する。
// 発生回数が多い都道府県から最大 10 件を個別に表示し、残りは「その他」に集約する。
func makeHourlyChartData(h domain.HourlyEarthquakeData) template.JS {
	jst := time.FixedZone("JST", 9*60*60)
	labels := make([]string, len(h.Hours))
	for i, t := range h.Hours {
		labels[i] = t.In(jst).Format("1/2 15:04")
	}

	type prefTotal struct {
		pref  domain.HourlyPrefectureCount
		total int
	}
	var pts []prefTotal
	for _, p := range h.Prefs {
		total := 0
		for _, c := range p.Counts {
			total += c
		}
		if total > 0 {
			pts = append(pts, prefTotal{p, total})
		}
	}
	sort.Slice(pts, func(i, j int) bool {
		return pts[i].total > pts[j].total
	})

	var datasets []hourlyDatasetJSON
	for i, pt := range pts {
		datasets = append(datasets, hourlyDatasetJSON{
			Label:           pt.pref.PrefName,
			Data:            pt.pref.Counts,
			BackgroundColor: hourlyChartColors[i%len(hourlyChartColors)],
			Stack:           "total",
		})
	}

	data := hourlyChartDataJSON{Labels: labels, Datasets: datasets}
	b, err := json.Marshal(data)
	if err != nil {
		return template.JS(`{"labels":[],"datasets":[]}`)
	}
	return template.JS(b)
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
		"Ranking":                displayData.RankingRecords,
		"LatestEarthquakes":      displayData.LatestEarthquakes,
		"TodayBigEarthquakes":    displayData.TodayBigEarthquakes,
		"WeekBigEarthquakes":     displayData.WeekBigEarthquakes,
		"MonthBigEarthquakes":    displayData.MonthBigEarthquakes,
		"TodayPrefectureRanking": displayData.TodayPrefectureRanking,
		"WeekPrefectureRanking":  displayData.WeekPrefectureRanking,
		"SurgeRanking":           displayData.SurgeRanking,
		"Summary":                displayData.Summary,
		"ReportPeriod":           from.Format("2006-01-02") + " ～ " + to.Format("2006-01-02"),
		"Now":                    now.Format(time.RFC3339),
		"HourlyChartData":        makeHourlyChartData(displayData.HourlyEarthquake),
	}
	if err := g.tmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}
