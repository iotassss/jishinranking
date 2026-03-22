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
	tmpl        *template.Template
	detailTmpl  *template.Template
	aboutTmpl   *template.Template
	historyTmpl *template.Template
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
	jst := time.FixedZone("JST", 9*60*60)

	// index.html
	tmplPath := filepath.Join(templatePath, "index.html")
	basePath := filepath.Join(templatePath, "_base.html")
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(basePath, tmplPath)
	if err != nil {
		panic(fmt.Sprintf("template parse error: %v", err))
	}

	// detail.html
	detailFuncMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"fmtTime": func(t time.Time, layout string) string {
			return t.In(jst).Format(layout)
		},
		"intensityClass": domain.IntensityPillClassFor,
		"joinStrings": func(ss []string, sep string) string {
			result := ""
			for i, s := range ss {
				if i > 0 {
					result += sep
				}
				result += s
			}
			return result
		},
		"prefName":   func(p domain.PrefIntensityDetail) string { return p.Name },
		"prefMaxInt": func(p domain.PrefIntensityDetail) string { return p.MaxInt },
	}
	detailTmplPath := filepath.Join(templatePath, "detail.html")
	detailTmpl, err := template.New("detail.html").Funcs(detailFuncMap).ParseFiles(basePath, detailTmplPath)
	if err != nil {
		panic(fmt.Sprintf("detail template parse error: %v", err))
	}

	// about.html
	aboutTmplPath := filepath.Join(templatePath, "about.html")
	aboutTmpl, err := template.New("about.html").ParseFiles(basePath, aboutTmplPath)
	if err != nil {
		panic(fmt.Sprintf("about template parse error: %v", err))
	}

	// history.html
	historyFuncMap := template.FuncMap{
		"fmtTime": func(t time.Time, layout string) string {
			return t.In(jst).Format(layout)
		},
		"intensityLevel": domain.IntensityLevel,
	}
	historyTmplPath := filepath.Join(templatePath, "history.html")
	historyTmpl, err := template.New("history.html").Funcs(historyFuncMap).ParseFiles(basePath, historyTmplPath)
	if err != nil {
		panic(fmt.Sprintf("history template parse error: %v", err))
	}

	return &SimpleHTMLGenerator{tmpl: tmpl, detailTmpl: detailTmpl, aboutTmpl: aboutTmpl, historyTmpl: historyTmpl}
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
		"BrandSub":               "都道府県別に地震をランキング＆可視化",
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
		"WeekTotalCount":         displayData.WeekReportCount,
	}
	if err := g.tmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}

// GenerateHistory は地震履歴ページのHTMLを生成する。
// tab は "6h", "today", "week", "month" のいずれか。
// counts は各タブの件数（タブナビの件数バッジ表示用）。
func (g *SimpleHTMLGenerator) GenerateHistory(
	tab string,
	earthquakes domain.EarthquakeRecordList,
	counts map[string]int,
	now time.Time,
) (domain.PublishedHTML, error) {
	jst := time.FixedZone("JST", 9*60*60)

	// デフォルト表示順: 震度降順 → M値降順 → 時刻降順
	sorted := make(domain.EarthquakeRecordList, len(earthquakes))
	copy(sorted, earthquakes)
	sort.Slice(sorted, func(i, j int) bool {
		li := domain.IntensityLevel(sorted[i].MaxIntensity)
		lj := domain.IntensityLevel(sorted[j].MaxIntensity)
		if li != lj {
			return li > lj
		}
		if sorted[i].Magnitude != sorted[j].Magnitude {
			return sorted[i].Magnitude > sorted[j].Magnitude
		}
		return sorted[i].OccurredAt.After(sorted[j].OccurredAt)
	})
	earthquakes = sorted

	// 統計計算
	totalCount := len(earthquakes)
	alertCount := 0
	maxMag := 0.0
	maxInt := "-"
	maxIntLevel := 0
	for _, eq := range earthquakes {
		if eq.Magnitude > maxMag {
			maxMag = eq.Magnitude
		}
		lv := domain.IntensityLevel(eq.MaxIntensity)
		if lv > maxIntLevel {
			maxIntLevel = lv
			maxInt = eq.MaxIntensity
		}
		if lv >= 3 {
			alertCount++
		}
	}

	// タブごとのラベル・タイトル
	tabLabel := map[string]string{
		"6h":    "過去6時間",
		"today": "今日（24時間）",
		"week":  "今週（7日間）",
		"month": "今月（30日間）",
	}

	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub":    "地震発生履歴",
		"Tab":         tab,
		"TabTitle":    tabLabel[tab],
		"Earthquakes": earthquakes,
		"Count6h":     counts["6h"],
		"CountToday":  counts["today"],
		"CountWeek":   counts["week"],
		"CountMonth":  counts["month"],
		"TotalCount":  totalCount,
		"AlertCount":  alertCount,
		"MaxMag":      maxMag,
		"MaxInt":      maxInt,
		"MaxIntLevel": maxIntLevel,
		"Now":         now.In(jst).Format(time.RFC3339),
		"NowTime":     now,
	}
	if err := g.historyTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("history template execute failed (tab=%s): %w", tab, err)
	}
	return domain.PublishedHTML(buf.String()), nil
}

// GenerateAbout はサイト概要ページのHTMLを生成する。
func (g *SimpleHTMLGenerator) GenerateAbout() (domain.PublishedHTML, error) {
	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub": "都道府県別に地震をランキング＆可視化",
	}
	if err := g.aboutTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("about template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}

// GenerateDetail は1地震イベントの詳細HTMLを生成する。
func (g *SimpleHTMLGenerator) GenerateDetail(detail domain.EarthquakeDetail) (domain.PublishedHTML, error) {
	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub":         "地震詳細",
		"EventID":          detail.EventID,
		"OccurredAt":       detail.OccurredAt,
		"ReportedAt":       detail.ReportedAt,
		"Status":           detail.Status,
		"PublishingOffice": detail.PublishingOffice,
		"HeadlineText":     detail.HeadlineText,
		"Hypocenter":       detail.Hypocenter,
		"Latitude":         detail.Latitude,
		"Longitude":        detail.Longitude,
		"Depth":            detail.Depth,
		"CoordinateKnown":  detail.CoordinateKnown,
		"Magnitude":        detail.Magnitude,
		"MagnitudeType":    detail.MagnitudeType,
		"MaxIntensity":     detail.MaxIntensity,
		"ObservedPrefs":    detail.ObservedPrefs,
		// script内で使う数値をtemplate.JSとして渡すことでエスケープを回避する
		"LatJS": template.JS(fmt.Sprintf("%g", detail.Latitude)),
		"LngJS": template.JS(fmt.Sprintf("%g", detail.Longitude)),
	}
	if err := g.detailTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("detail template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}
