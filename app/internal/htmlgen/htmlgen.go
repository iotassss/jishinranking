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
	tmpl            *template.Template
	detailTmpl      *template.Template
	aboutTmpl       *template.Template
	historyTmpl     *template.Template
	prefTmpl        *template.Template
	prefRankingTmpl *template.Template
	weekScoreTmpl   *template.Template
}

// ---- hourly chart helpers (散布図) ----

type scatterPointJSON struct {
	X int64   `json:"x"` // Unix ms
	Y float64 `json:"y"` // マグニチュード
}

type scatterDatasetJSON struct {
	Label                string             `json:"label"`
	Data                 []scatterPointJSON `json:"data"`
	PointStyle           string             `json:"pointStyle"`
	PointRadius          []int              `json:"pointRadius"`
	PointBorderWidth     []int              `json:"pointBorderWidth"`
	PointBorderColor     []string           `json:"pointBorderColor"`
	PointBackgroundColor []string           `json:"pointBackgroundColor"`
}

// intensityStyle はX印の見た目パラメータ（震度レベル 0〜9 に対応）
type intensityStyle struct {
	radius      int
	borderWidth int
	color       string // rgba()
}

// intensityStyleMap[i] は IntensityLevel()==i のときに使うスタイル
// 震度小: 小さく薄く　震度大: 大きく濃く
var intensityStyleMap = [10]intensityStyle{
	{3, 1, "rgba(180,180,180,0.30)"}, // 0: -（不明）
	{3, 1, "rgba(100,149,237,0.50)"}, // 1: 震度1
	{4, 1, "rgba(100,149,237,0.62)"}, // 2: 震度2
	{5, 1, "rgba(67,99,216,0.70)"},   // 3: 震度3
	{6, 2, "rgba(67,99,216,0.80)"},   // 4: 震度4
	{8, 2, "rgba(220,140,0,0.83)"},   // 5: 震度5弱
	{9, 2, "rgba(220,90,0,0.87)"},    // 6: 震度5強
	{11, 3, "rgba(200,30,0,0.91)"},   // 7: 震度6弱
	{13, 3, "rgba(170,0,0,0.95)"},    // 8: 震度6強
	{15, 4, "rgba(120,0,0,1.00)"},    // 9: 震度7
}

type scatterChartDataJSON struct {
	Datasets []scatterDatasetJSON `json:"datasets"`
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

// makeHourlyChartData は HourlyEarthquakeData を Chart.js 散布図用 JSON に変換する。
// 各地震を (発生時刻 Unix ms, マグニチュード) の点として描画し、
// 最大震度が大きいほど X 印を大きく・太く・濃くする。
func makeHourlyChartData(h domain.HourlyEarthquakeData) template.JS {
	n := len(h.Points)
	pts := make([]scatterPointJSON, n)
	radii := make([]int, n)
	bws := make([]int, n)
	colors := make([]string, n)

	for i, p := range h.Points {
		pts[i] = scatterPointJSON{X: p.X, Y: p.Y}
		level := domain.IntensityLevel(p.Intensity)
		if level < 0 {
			level = 0
		} else if level >= len(intensityStyleMap) {
			level = len(intensityStyleMap) - 1
		}
		s := intensityStyleMap[level]
		radii[i] = s.radius
		bws[i] = s.borderWidth
		colors[i] = s.color
	}

	data := scatterChartDataJSON{
		Datasets: []scatterDatasetJSON{
			{
				Label:                "地震",
				Data:                 pts,
				PointStyle:           "crossRot",
				PointRadius:          radii,
				PointBorderWidth:     bws,
				PointBorderColor:     colors,
				PointBackgroundColor: colors,
			},
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		return template.JS(`{"datasets":[]}`)
	}
	return template.JS(b)
}

// makePrefScatterChartData は EpicenterPoint リストを Chart.js 散布図用 JSON に変換する。
// intensityStyleMap の色・サイズはその都道府県での観測震度 (PrefMaxInt) をもとに決定する。
func makePrefScatterChartData(epicenters []domain.EpicenterPoint) template.JS {
	n := len(epicenters)
	pts := make([]scatterPointJSON, n)
	radii := make([]int, n)
	bws := make([]int, n)
	colors := make([]string, n)

	for i, ep := range epicenters {
		pts[i] = scatterPointJSON{X: ep.OccurredAt.UnixMilli(), Y: ep.Magnitude}
		level := domain.IntensityLevel(ep.PrefMaxInt)
		if level < 0 {
			level = 0
		} else if level >= len(intensityStyleMap) {
			level = len(intensityStyleMap) - 1
		}
		s := intensityStyleMap[level]
		radii[i] = s.radius
		bws[i] = s.borderWidth
		colors[i] = s.color
	}

	chartData := scatterChartDataJSON{
		Datasets: []scatterDatasetJSON{
			{
				Label:                "地震",
				Data:                 pts,
				PointStyle:           "crossRot",
				PointRadius:          radii,
				PointBorderWidth:     bws,
				PointBorderColor:     colors,
				PointBackgroundColor: colors,
			},
		},
	}
	b, err := json.Marshal(chartData)
	if err != nil {
		return template.JS(`{"datasets":[]}`)
	}
	return template.JS(b)
}

func NewSimpleHTMLGenerator(templatePath string) *SimpleHTMLGenerator {
	jst := time.FixedZone("JST", 9*60*60)

	// index.html
	tmplPath := filepath.Join(templatePath, "index.html")
	basePath := filepath.Join(templatePath, "_base.html")
	funcMap := template.FuncMap{
		"add":              func(a, b int) int { return a + b },
		"mul":              func(a, b float64) float64 { return a * b },
		"intensityDisplay": domain.IntensityDisplay,
		"intensityClass":   domain.IntensityPillClassFor,
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(basePath, tmplPath)
	if err != nil {
		panic(fmt.Sprintf("template parse error: %v", err))
	}

	// detail.html
	detailFuncMap := template.FuncMap{
		"add":              func(a, b int) int { return a + b },
		"intensityDisplay": domain.IntensityDisplay,
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
		"intensityLevel":   domain.IntensityLevel,
		"intensityDisplay": domain.IntensityDisplay,
	}
	historyTmplPath := filepath.Join(templatePath, "history.html")
	historyTmpl, err := template.New("history.html").Funcs(historyFuncMap).ParseFiles(basePath, historyTmplPath)
	if err != nil {
		panic(fmt.Sprintf("history template parse error: %v", err))
	}

	// pref.html
	prefFuncMap := template.FuncMap{
		"fmtTime": func(t time.Time, layout string) string {
			return t.In(jst).Format(layout)
		},
		"intensityClass":   domain.IntensityPillClassFor,
		"intensityDisplay": domain.IntensityDisplay,
		"prefMaxIntFor": func(eq domain.EarthquakeRecord, prefCode string) string {
			for _, p := range eq.ObservedPrefs {
				if p.Code == prefCode {
					return p.MaxInt
				}
			}
			return "-"
		},
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"addf": func(a, b float64) float64 { return a + b },
		"itof": func(i int) float64 { return float64(i) },
		"not":  func(b bool) bool { return !b },
	}
	prefTmplPath := filepath.Join(templatePath, "pref.html")
	prefTmpl, err := template.New("pref.html").Funcs(prefFuncMap).ParseFiles(basePath, prefTmplPath)
	if err != nil {
		panic(fmt.Sprintf("pref template parse error: %v", err))
	}

	// pref_ranking.html
	prefRankingFuncMap := template.FuncMap{
		"fmtTime": func(t time.Time, layout string) string {
			return t.In(jst).Format(layout)
		},
		"intensityDisplay": domain.IntensityDisplay,
		"intensityLevel":   domain.IntensityLevel,
	}
	prefRankingTmplPath := filepath.Join(templatePath, "pref_ranking.html")
	prefRankingTmpl, err := template.New("pref_ranking.html").Funcs(prefRankingFuncMap).ParseFiles(basePath, prefRankingTmplPath)
	if err != nil {
		panic(fmt.Sprintf("pref_ranking template parse error: %v", err))
	}

	// week_score.html
	weekScoreFuncMap := template.FuncMap{
		"fmtTime": func(t time.Time, layout string) string {
			return t.In(jst).Format(layout)
		},
		"scoreBarWidth": func(score, maxScore float64) int {
			if maxScore <= 0 {
				return 0
			}
			w := int(score / maxScore * 80)
			if w < 2 {
				w = 2
			}
			return w
		},
	}
	weekScoreTmplPath := filepath.Join(templatePath, "week_score.html")
	weekScoreTmpl, err := template.New("week_score.html").Funcs(weekScoreFuncMap).ParseFiles(basePath, weekScoreTmplPath)
	if err != nil {
		panic(fmt.Sprintf("week_score template parse error: %v", err))
	}

	return &SimpleHTMLGenerator{tmpl: tmpl, detailTmpl: detailTmpl, aboutTmpl: aboutTmpl, historyTmpl: historyTmpl, prefTmpl: prefTmpl, prefRankingTmpl: prefRankingTmpl, weekScoreTmpl: weekScoreTmpl}
}

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
		"WeekScoreRanking":       displayData.WeekScoreRanking,
		"WeekScoreRankingAll":    displayData.WeekScoreRankingAll,
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

// GeneratePref は都道府県別ページのHTMLを生成する。
func (g *SimpleHTMLGenerator) GeneratePref(data domain.PrefPageData) (domain.PublishedHTML, error) {
	jst := time.FixedZone("JST", 9*60*60)

	// 1時間ごとのChart.js用ラベルを生成（日付が変わる0時のみ日付、それ以外は空文字）
	hourlyLabels := make([]string, len(data.HourlyHours))
	hourlyTooltipLabels := make([]string, len(data.HourlyHours))
	for i, t := range data.HourlyHours {
		jt := t.In(jst)
		if jt.Hour() == 0 {
			hourlyLabels[i] = jt.Format("1/2")
		} else {
			hourlyLabels[i] = ""
		}
		hourlyTooltipLabels[i] = jt.Format("1/2 15:04") + "〜"
	}

	// JSON化（template.JS でエスケープなし出力）
	labelsJSON, _ := json.Marshal(hourlyLabels)
	countsJSON, _ := json.Marshal(data.HourlyCounts)
	tooltipJSON, _ := json.Marshal(hourlyTooltipLabels)

	// 震源地マップ用JSON: [lat, lng, magnitude, 観測震度, 震源名, 時刻文字列, URL]
	type epicenterEntry [7]interface{}
	epicenterEntries := make([]epicenterEntry, 0, len(data.Epicenters))
	for _, ep := range data.Epicenters {
		epicenterEntries = append(epicenterEntries, epicenterEntry{
			ep.Lat,
			ep.Lng,
			ep.Magnitude,
			ep.PrefMaxInt,
			ep.Hypocenter,
			ep.OccurredAt.In(jst).Format("2006-01-02 15:04"),
			ep.DetailURL,
		})
	}
	epicenterJSON, _ := json.Marshal(epicenterEntries)

	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub":              data.PrefName + "の地震情報",
		"PrefCode":              data.PrefCode,
		"PrefName":              data.PrefName,
		"WeekRank":              data.WeekRank,
		"WeekCount":             data.WeekCount,
		"WeekRatio":             data.WeekRatio,
		"WeekAvg":               data.WeekAvg,
		"WeekScoreRank":         data.WeekScoreRank,
		"WeekScore":             data.WeekScore,
		"MaxIntensity":          data.MaxIntensity,
		"MaxIntensityAt":        data.MaxIntensityAt,
		"TodayCount":            data.TodayCount,
		"SurgeScore":            data.SurgeScore,
		"SurgeRank":             data.SurgeRank,
		"NationalAvgRatio":      data.NationalAvgRatio,
		"RecentEarthquakes":     data.RecentEarthquakes,
		"HourlyLabelsJS":        template.JS(labelsJSON),
		"HourlyCountsJS":        template.JS(countsJSON),
		"HourlyTooltipLabelsJS": template.JS(tooltipJSON),
		"EpicenterJS":           template.JS(epicenterJSON),
		"PrefHourlyChartData":   makePrefScatterChartData(data.Epicenters),
		"UpdatedAt":             data.UpdatedAt,
		"WeekFrom":              data.WeekFrom,
		"WeekTo":                data.WeekTo,
	}
	if err := g.prefTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("pref template execute failed (pref=%s): %w", data.PrefCode, err)
	}
	return domain.PublishedHTML(buf.String()), nil
}

// GeneratePrefRanking は全47都道府県ランキングページのHTMLを生成する。
func (g *SimpleHTMLGenerator) GeneratePrefRanking(
	todayRanking domain.RankingRecordList,
	weekRanking domain.RankingRecordList,
	weekScoreRanking domain.WeekScoreRecordList,
	now time.Time,
) (domain.PublishedHTML, error) {
	jst := time.FixedZone("JST", 9*60*60)

	// 都道府県コード → WeekScore のマップを構築
	scoreMap := make(map[string]float64, len(weekScoreRanking))
	for _, r := range weekScoreRanking {
		scoreMap[r.PrefCode] = r.WeekScore
	}

	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub":     "都道府県別 地震ランキング",
		"TodayRanking": todayRanking,
		"WeekRanking":  weekRanking,
		"ScoreMap":     scoreMap,
		"Now":          now.In(jst).Format(time.RFC3339),
		"NowTime":      now,
	}
	if err := g.prefRankingTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("pref_ranking template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}

// GenerateWeekScore は今週の都道府県別地震スコアページのHTMLを生成する。
func (g *SimpleHTMLGenerator) GenerateWeekScore(
	ranking domain.WeekScoreRecordList,
	minShortScore float64,
	minRatio float64,
	now time.Time,
) (domain.PublishedHTML, error) {
	jst := time.FixedZone("JST", 9*60*60)
	// スコアバー描画用に最大スコアを計算
	var maxScore, maxShortScore float64
	for _, r := range ranking {
		if r.WeekScore > maxScore {
			maxScore = r.WeekScore
		}
		if r.ShortScore > maxShortScore {
			maxShortScore = r.ShortScore
		}
	}
	var buf bytes.Buffer
	dataMap := map[string]interface{}{
		"BrandSub":         "今週の都道府県別地震スコア",
		"WeekScoreRanking": ranking,
		"MaxScore":         maxScore,
		"MaxShortScore":    maxShortScore,
		"MinShortScore":    minShortScore,
		"MinRatio":         minRatio,
		"Now":              now.In(jst).Format(time.RFC3339),
		"NowTime":          now,
	}
	if err := g.weekScoreTmpl.Execute(&buf, dataMap); err != nil {
		return "", fmt.Errorf("week_score template execute failed: %w", err)
	}
	return domain.PublishedHTML(buf.String()), nil
}
