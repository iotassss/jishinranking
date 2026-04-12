// cmd/gendata は、開発・テスト用のサンプルJSONデータを生成するスクリプトです。
//
// 生成ルール:
//   - 長崎県: 現在時刻に大きな地震（5+）が発生、過去にも数件の背景地震あり
//   - 茨城県: 7日前〜1.5日前に1件の微小地震（震度1）、
//     1.5日前〜現在までに5件の小地震（震度2）→ 急上昇倍率 R≈2.24
//   - その他の都道府県: 7日以内のランダムな日時で発生
//
// 使用方法:
//
//	go run ./cmd/gendata > tmp/testdata/$(date -u +%Y%m%dT%H%M%SZ).json
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"
)

// ---------------------------------------------------------------------------
// 時刻型（JSONシリアライズ時のタイムゾーン制御）
// ---------------------------------------------------------------------------

// utcTime は control.dateTime 用。常に UTC ("Z") で出力する。
type utcTime struct{ time.Time }

func (t utcTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Truncate(time.Second).Format(time.RFC3339))
}

// jstTime は reportDateTime / originTime 用。JST (+09:00) で出力する。
type jstTime struct{ time.Time }

func (t jstTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.In(jst()).Truncate(time.Second).Format(time.RFC3339))
}

func jst() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	return loc
}

// ---------------------------------------------------------------------------
// JSON スキーマ定義（domain.Report と同じ構造、json タグのみ）
// ---------------------------------------------------------------------------

type Report struct {
	Control  Control  `json:"control"`
	Head     Head     `json:"head"`
	Body     Body     `json:"body"`
	Metadata Metadata `json:"metadata"`
}

type Control struct {
	Title            string  `json:"title"`
	DateTime         utcTime `json:"dateTime"`
	Status           string  `json:"status"`
	EditorialOffice  string  `json:"editorialOffice"`
	PublishingOffice string  `json:"publishingOffice"`
}

type Head struct {
	Title           string   `json:"title"`
	ReportDateTime  jstTime  `json:"reportDateTime"`
	TargetDateTime  jstTime  `json:"targetDateTime"`
	EventID         string   `json:"eventID"`
	InfoType        string   `json:"infoType"`
	Serial          string   `json:"serial"`
	InfoKind        string   `json:"infoKind"`
	InfoKindVersion string   `json:"infoKindVersion"`
	Headline        Headline `json:"headline"`
}

type Headline struct {
	Text         string        `json:"text"`
	Informations []interface{} `json:"informations"` // null
}

type Body struct {
	Earthquake *Earthquake `json:"earthquake"`
	Intensity  *Intensity  `json:"intensity"`
	Comments   *Comments   `json:"comments"`
}

type Earthquake struct {
	OriginTime  jstTime     `json:"originTime"`
	ArrivalTime jstTime     `json:"arrivalTime"`
	Condition   string      `json:"condition"`
	Hypocenter  *Hypocenter `json:"hypocenter"`
	Magnitude   *Magnitude  `json:"magnitude"`
}

type Hypocenter struct {
	Area     *HypocenterArea `json:"area"`
	Location interface{}     `json:"location"` // null
}

type HypocenterArea struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	Coordinate string `json:"coordinate"`
}

type Magnitude struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Value       string `json:"value"`
}

type Intensity struct {
	Observation *Observation `json:"observation"`
}

type Observation struct {
	CodeDefine *CodeDefine `json:"codeDefine"`
	MaxInt     string      `json:"maxInt"`
	Prefs      []Pref      `json:"prefs"`
}

type CodeDefine struct {
	Types []CodeType `json:"types"`
}

type CodeType struct {
	XPath string `json:"xpath"`
	Name  string `json:"name"`
}

type Pref struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	MaxInt string `json:"maxInt"`
	Areas  []Area `json:"areas"`
}

type Area struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	MaxInt string `json:"maxInt"`
	Cities []City `json:"cities"`
}

type City struct {
	Name              string    `json:"name"`
	Code              string    `json:"code"`
	MaxInt            string    `json:"maxInt"`
	IntensityStations []Station `json:"intensityStations"`
}

type Station struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Int  string `json:"int"`
}

type Comments struct {
	ForecastComment *Comment `json:"forecastComment"`
	VarComment      *Comment `json:"varComment"`
}

type Comment struct {
	CodeType string `json:"codeType"`
	Text     string `json:"text"`
	Code     string `json:"code"`
}

type Metadata struct {
	URL string `json:"url"`
}

// ---------------------------------------------------------------------------
// 共通ヘルパー
// ---------------------------------------------------------------------------

func defaultCodeDefine() *CodeDefine {
	return &CodeDefine{
		Types: []CodeType{
			{XPath: "Pref/Code", Name: "地震情報／都道府県等"},
			{XPath: "Pref/Area/Code", Name: "地震情報／細分区域"},
			{XPath: "Pref/Area/City/Code", Name: "気象・地震・火山情報／市町村等"},
			{XPath: "Pref/Area/City/IntensityStation/Code", Name: "震度観測点"},
		},
	}
}

func defaultComments() *Comments {
	return &Comments{
		ForecastComment: &Comment{
			CodeType: "固定付加文",
			Text:     "この地震による津波の心配はありません。",
			Code:     "0215",
		},
		VarComment: &Comment{
			CodeType: "固定付加文",
			Text:     "印は気象庁以外の震度観測点についての情報です。",
			Code:     "0262",
		},
	}
}

// ---------------------------------------------------------------------------
// 地震仕様定義
// ---------------------------------------------------------------------------

// quakeSpec は1件の地震レポートを生成するためのパラメータ
type quakeSpec struct {
	OccurredAt  time.Time
	PrefName    string
	PrefCode    string
	AreaName    string
	AreaCode    string
	HypName     string
	HypCode     string
	HypCoord    string
	Office      string
	CityName    string
	CityCode    string
	StationName string
	StationCode string
	MaxInt      string
	Magnitude   float64
}

func buildReport(s quakeSpec) Report {
	t := s.OccurredAt.In(jst()).Truncate(time.Second)
	utc := t.UTC()
	magStr := fmt.Sprintf("%.1f", s.Magnitude)
	eventID := t.Format("200601021504")

	return Report{
		Control: Control{
			Title:            "震源・震度に関する情報",
			DateTime:         utcTime{utc},
			Status:           "通常",
			EditorialOffice:  s.Office,
			PublishingOffice: "気象庁",
		},
		Head: Head{
			Title:           "震源・震度情報",
			ReportDateTime:  jstTime{t},
			TargetDateTime:  jstTime{t},
			EventID:         eventID,
			InfoType:        "発表",
			Serial:          "1",
			InfoKind:        "地震情報",
			InfoKindVersion: "1.0_1",
			Headline: Headline{
				Text: fmt.Sprintf(
					"　%d日%02d時%02d分ころ、地震がありました。",
					t.Day(), t.Hour(), t.Minute(),
				),
				Informations: nil,
			},
		},
		Body: Body{
			Earthquake: &Earthquake{
				OriginTime:  jstTime{t},
				ArrivalTime: jstTime{t},
				Condition:   "",
				Hypocenter: &Hypocenter{
					Area: &HypocenterArea{
						Name:       s.HypName,
						Code:       s.HypCode,
						Coordinate: s.HypCoord,
					},
					Location: nil,
				},
				Magnitude: &Magnitude{
					Type:        "Mj",
					Description: "M" + magStr,
					Value:       magStr,
				},
			},
			Intensity: &Intensity{
				Observation: &Observation{
					CodeDefine: defaultCodeDefine(),
					MaxInt:     s.MaxInt,
					Prefs: []Pref{
						{
							Name:   s.PrefName,
							Code:   s.PrefCode,
							MaxInt: s.MaxInt,
							Areas: []Area{
								{
									Name:   s.AreaName,
									Code:   s.AreaCode,
									MaxInt: s.MaxInt,
									Cities: []City{
										{
											Name:   s.CityName,
											Code:   s.CityCode,
											MaxInt: s.MaxInt,
											IntensityStations: []Station{
												{
													Name: s.StationName,
													Code: s.StationCode,
													Int:  s.MaxInt,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Comments: defaultComments(),
		},
		Metadata: Metadata{
			URL: fmt.Sprintf(
				"https://www.data.jma.go.jp/developer/xml/data/%s_0_VXSE53_470000.xml",
				utc.Format("20060102150405"),
			),
		},
	}
}

// prefSpec は buildMultiPrefReport で使う都道府県1件分のパラメータ。
type prefSpec struct {
	PrefName    string
	PrefCode    string
	AreaName    string
	AreaCode    string
	CityName    string
	CityCode    string
	StationName string
	StationCode string
	MaxInt      string
}

// buildMultiPrefReport は複数都道府県に震度観測がある地震レポートを生成する。
// 先頭 prefSpec の MaxInt がレポート全体の最大震度になる。
func buildMultiPrefReport(t time.Time, hypName, hypCode, hypCoord, office string, mag float64, prefs []prefSpec) Report {
	t = t.In(jst()).Truncate(time.Second)
	utc := t.UTC()
	magStr := fmt.Sprintf("%.1f", mag)
	eventID := t.Format("200601021504")

	maxInt := prefs[0].MaxInt
	obsPrefs := make([]Pref, len(prefs))
	for i, p := range prefs {
		obsPrefs[i] = Pref{
			Name:   p.PrefName,
			Code:   p.PrefCode,
			MaxInt: p.MaxInt,
			Areas: []Area{
				{
					Name:   p.AreaName,
					Code:   p.AreaCode,
					MaxInt: p.MaxInt,
					Cities: []City{
						{
							Name:   p.CityName,
							Code:   p.CityCode,
							MaxInt: p.MaxInt,
							IntensityStations: []Station{
								{Name: p.StationName, Code: p.StationCode, Int: p.MaxInt},
							},
						},
					},
				},
			},
		}
	}

	return Report{
		Control: Control{
			Title:            "震源・震度に関する情報",
			DateTime:         utcTime{utc},
			Status:           "通常",
			EditorialOffice:  office,
			PublishingOffice: "気象庁",
		},
		Head: Head{
			Title:           "震源・震度情報",
			ReportDateTime:  jstTime{t},
			TargetDateTime:  jstTime{t},
			EventID:         eventID,
			InfoType:        "発表",
			Serial:          "1",
			InfoKind:        "地震情報",
			InfoKindVersion: "1.0_1",
			Headline: Headline{
				Text: fmt.Sprintf(
					"　%d日%02d時%02d分ころ、地震がありました。",
					t.Day(), t.Hour(), t.Minute(),
				),
				Informations: nil,
			},
		},
		Body: Body{
			Earthquake: &Earthquake{
				OriginTime:  jstTime{t},
				ArrivalTime: jstTime{t},
				Condition:   "",
				Hypocenter: &Hypocenter{
					Area: &HypocenterArea{
						Name:       hypName,
						Code:       hypCode,
						Coordinate: hypCoord,
					},
					Location: nil,
				},
				Magnitude: &Magnitude{
					Type:        "Mj",
					Description: "M" + magStr,
					Value:       magStr,
				},
			},
			Intensity: &Intensity{
				Observation: &Observation{
					CodeDefine: defaultCodeDefine(),
					MaxInt:     maxInt,
					Prefs:      obsPrefs,
				},
			},
			Comments: defaultComments(),
		},
		Metadata: Metadata{
			URL: fmt.Sprintf(
				"https://www.data.jma.go.jp/developer/xml/data/%s_0_VXSE53_470000.xml",
				utc.Format("20060102150405"),
			),
		},
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	loc := jst()
	now := time.Now().In(loc)
	rng := rand.New(rand.NewSource(now.UnixNano()))

	// now から指定した時間範囲内でランダムな時刻を返す
	randBetween := func(minHours, maxHours float64) time.Time {
		h := minHours + rng.Float64()*(maxHours-minHours)
		return now.Add(-time.Duration(h * float64(time.Hour))).Truncate(time.Minute)
	}

	// now から d 日前の時刻を返す
	daysAgo := func(d float64) time.Time {
		return now.Add(-time.Duration(d * 24 * float64(time.Hour))).Truncate(time.Minute)
	}

	// jitterCoord は基準緯度経度を ±0.2° 揺らした座標文字列を返す
	jitterCoord := func(latBase, lonBase float64, depthM int) string {
		lat := latBase + (rng.Float64()-0.5)*0.4
		lon := lonBase + (rng.Float64()-0.5)*0.4
		latSign, lonSign := "+", "+"
		if lat < 0 {
			latSign = ""
		}
		if lon < 0 {
			lonSign = ""
		}
		return fmt.Sprintf("%s%.1f%s%.1f-%d/", latSign, lat, lonSign, lon, depthM)
	}

	// 茨城県用テンプレート（coord は jitterCoord で毎回生成）
	ibarakiSpec := func(t time.Time, coord, maxInt, cityName, cityCode, stationName, stationCode string, mag float64) quakeSpec {
		return quakeSpec{
			OccurredAt:  t,
			PrefName:    "茨城県",
			PrefCode:    "08",
			AreaName:    "茨城県南部",
			AreaCode:    "301",
			HypName:     "茨城県南部",
			HypCode:     "301",
			HypCoord:    coord,
			Office:      "気象庁本庁",
			CityName:    cityName,
			CityCode:    cityCode,
			StationName: stationName,
			StationCode: stationCode,
			MaxInt:      maxInt,
			Magnitude:   mag,
		}
	}

	// 長崎県用テンプレート（coord は jitterCoord で毎回生成）
	nagasakiSpec := func(t time.Time, coord, maxInt, cityName, cityCode, stationName, stationCode string, mag float64) quakeSpec {
		return quakeSpec{
			OccurredAt:  t,
			PrefName:    "長崎県",
			PrefCode:    "42",
			AreaName:    "長崎県南部",
			AreaCode:    "720",
			HypName:     "長崎県南西部",
			HypCode:     "730",
			HypCoord:    coord,
			Office:      "福岡管区気象台",
			CityName:    cityName,
			CityCode:    cityCode,
			StationName: stationName,
			StationCode: stationCode,
			MaxInt:      maxInt,
			Magnitude:   mag,
		}
	}

	// ===== 静岡県（遠州灘）M7.5 大地震 =====
	// 静岡7 / 愛知6+ / 三重6- / 長野5+ / 岐阜5-
	shizuokaQuake := buildMultiPrefReport(
		daysAgo(0.03), // 約43分前
		"遠州灘", "481", "+34.7+137.8-200/",
		"気象庁本庁", 7.5,
		[]prefSpec{
			{"静岡県", "22", "静岡県中部", "440", "静岡市葵区", "2210100", "静岡市葵区役所", "2210101", "7"},
			{"愛知県", "23", "愛知県西部", "490", "名古屋市中区", "2310100", "名古屋気象台", "2310101", "6+"},
			{"三重県", "24", "三重県北部", "500", "津市", "2420200", "津地方気象台", "2420201", "6-"},
			{"長野県", "20", "長野県南部", "380", "飯田市", "2020600", "飯田市役所", "2020601", "5+"},
			{"岐阜県", "21", "岐阜県南部", "400", "岐阜市", "2120100", "岐阜市役所", "2120101", "5-"},
		},
	)

	specs := []quakeSpec{
		// ===== 長崎県 =====
		// 現在時刻: メインの大きな地震（震度5+）
		nagasakiSpec(
			now.Truncate(time.Second),
			jitterCoord(32.4, 129.3, 10000),
			"5+", "大村市", "4220300", "大村市役所", "4220301", 5.0,
		),
		// 背景地震1: 2〜4日前（震度4）
		nagasakiSpec(
			randBetween(2*24, 4*24),
			jitterCoord(32.4, 129.3, 10000),
			"4", "長崎市", "4220100", "長崎市役所", "4220101", 3.5,
		),
		// 背景地震2: 4〜7日前（震度3）
		nagasakiSpec(
			randBetween(4*24, 7*24),
			jitterCoord(32.4, 129.3, 10000),
			"3", "長崎市", "4220100", "長崎市役所", "4220101", 3.0,
		),

		// ===== 茨城県 =====
		// 急上昇倍率 R≈2.24 となるよう設計:
		//   背景:  t=4.0d 前, 震度1(weight=1) × 1件
		//   直近:  t=0.05/0.2/0.6/1.0/1.4d 前, 震度2(weight=2) × 5件
		//
		// R = G_short / (G_long × τ_s/τ_l + 1)
		//   G_short ≈ 6.91, normalizedLong ≈ 2.08
		//   R = 6.91 / (2.08 + 1) ≈ 2.24
		ibarakiSpec(daysAgo(4.0), jitterCoord(36.1, 140.1, 50000),
			"1", "土浦市", "0820300", "土浦市常名", "0820301", 2.5),
		ibarakiSpec(daysAgo(1.4), jitterCoord(36.1, 140.1, 50000),
			"2", "水戸市", "0820100", "水戸市金町", "0820101", 2.7),
		ibarakiSpec(daysAgo(1.0), jitterCoord(36.1, 140.1, 50000),
			"2", "水戸市", "0820100", "水戸市金町", "0820101", 2.9),
		ibarakiSpec(daysAgo(0.6), jitterCoord(36.1, 140.1, 50000),
			"2", "水戸市", "0820100", "水戸市金町", "0820101", 3.0),
		ibarakiSpec(daysAgo(0.2), jitterCoord(36.1, 140.1, 50000),
			"2", "水戸市", "0820100", "水戸市金町", "0820101", 2.8),
		ibarakiSpec(daysAgo(0.05), jitterCoord(36.1, 140.1, 50000),
			"2", "水戸市", "0820100", "水戸市金町", "0820101", 2.6),

		// ===== 千葉県 =====
		{
			OccurredAt: randBetween(24, 5*24),
			PrefName:   "千葉県", PrefCode: "12",
			AreaName: "千葉県北西部", AreaCode: "461",
			HypName: "千葉県北西部", HypCode: "461", HypCoord: jitterCoord(35.6, 139.9, 70000),
			Office:   "気象庁本庁",
			CityName: "松戸市", CityCode: "1220300",
			StationName: "松戸市松戸", StationCode: "1220301",
			MaxInt: "2", Magnitude: 3.2,
		},
		{
			OccurredAt: randBetween(3*24, 7*24),
			PrefName:   "千葉県", PrefCode: "12",
			AreaName: "千葉県北西部", AreaCode: "461",
			HypName: "千葉県北西部", HypCode: "461", HypCoord: jitterCoord(35.6, 139.9, 70000),
			Office:   "気象庁本庁",
			CityName: "松戸市", CityCode: "1220300",
			StationName: "松戸市松戸", StationCode: "1220301",
			MaxInt: "1", Magnitude: 2.8,
		},

		// ===== 神奈川県 =====
		{
			OccurredAt: randBetween(24, 7*24),
			PrefName:   "神奈川県", PrefCode: "14",
			AreaName: "神奈川県西部", AreaCode: "470",
			HypName: "神奈川県西部", HypCode: "470", HypCoord: jitterCoord(35.3, 139.1, 20000),
			Office:   "気象庁本庁",
			CityName: "小田原市", CityCode: "1420700",
			StationName: "小田原市役所", StationCode: "1420701",
			MaxInt: "1", Magnitude: 2.9,
		},

		// ===== 東京都 =====
		{
			OccurredAt: randBetween(2*24, 7*24),
			PrefName:   "東京都", PrefCode: "13",
			AreaName: "東京都23区", AreaCode: "455",
			HypName: "東京都23区", HypCode: "455", HypCoord: jitterCoord(35.7, 139.7, 80000),
			Office:   "気象庁本庁",
			CityName: "江戸川区", CityCode: "1313500",
			StationName: "江戸川区葛西", StationCode: "1313501",
			MaxInt: "1", Magnitude: 2.9,
		},

		// ===== 熊本県 =====
		{
			OccurredAt: randBetween(0, 3*24),
			PrefName:   "熊本県", PrefCode: "43",
			AreaName: "熊本県南部", AreaCode: "690",
			HypName: "熊本県熊本地方", HypCode: "691", HypCoord: jitterCoord(32.8, 130.7, 10000),
			Office:   "福岡管区気象台",
			CityName: "熊本市", CityCode: "4310100",
			StationName: "熊本市中央区", StationCode: "4310101",
			MaxInt: "3", Magnitude: 3.5,
		},
		{
			OccurredAt: randBetween(3*24, 7*24),
			PrefName:   "熊本県", PrefCode: "43",
			AreaName: "熊本県南部", AreaCode: "690",
			HypName: "熊本県熊本地方", HypCode: "691", HypCoord: jitterCoord(32.8, 130.7, 10000),
			Office:   "福岡管区気象台",
			CityName: "熊本市", CityCode: "4310100",
			StationName: "熊本市中央区", StationCode: "4310101",
			MaxInt: "2", Magnitude: 3.1,
		},

		// ===== 福岡県 =====
		{
			OccurredAt: randBetween(24, 7*24),
			PrefName:   "福岡県", PrefCode: "40",
			AreaName: "福岡県北西部", AreaCode: "680",
			HypName: "福岡県北西部", HypCode: "680", HypCoord: jitterCoord(33.6, 130.3, 10000),
			Office:   "福岡管区気象台",
			CityName: "福岡市", CityCode: "4013100",
			StationName: "福岡市中央区", StationCode: "4013101",
			MaxInt: "1", Magnitude: 2.8,
		},

		// ===== 鹿児島県 =====
		{
			OccurredAt: randBetween(24, 7*24),
			PrefName:   "鹿児島県", PrefCode: "46",
			AreaName: "鹿児島県薩摩地方", AreaCode: "751",
			HypName: "鹿児島県薩摩地方", HypCode: "751", HypCoord: jitterCoord(31.5, 130.5, 10000),
			Office:   "福岡管区気象台",
			CityName: "薩摩川内市", CityCode: "4621900",
			StationName: "薩摩川内市役所", StationCode: "4621901",
			MaxInt: "2", Magnitude: 3.2,
		},
	}

	// レポートを生成
	reports := make([]Report, len(specs))
	for i, s := range specs {
		reports[i] = buildReport(s)
	}
	reports = append(reports, shizuokaQuake)

	// 発生時刻の降順（新しい順）でソート
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Body.Earthquake.OriginTime.Time.After(
			reports[j].Body.Earthquake.OriginTime.Time,
		)
	})

	// JSON 出力（HTML エスケープなし、インデントあり）
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(reports); err != nil {
		fmt.Fprintln(os.Stderr, "JSON encode error:", err)
		os.Exit(1)
	}
}
