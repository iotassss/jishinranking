package domain

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EarthquakeDetail は1地震イベントの詳細ページ用データ。
type EarthquakeDetail struct {
	EventID          string
	OccurredAt       time.Time
	ReportedAt       time.Time
	Status           string
	PublishingOffice string
	HeadlineText     string
	Hypocenter       string
	Latitude         float64
	Longitude        float64
	Depth            int  // km（0 かつ CoordinateKnown==false のとき不明）
	CoordinateKnown  bool // 座標が正常にパースできた場合 true
	Magnitude        float64
	MagnitudeType    string // "Mj" など
	MaxIntensity     string // "1"〜"7"（"-" のとき不明）
	ObservedPrefs    []PrefIntensityDetail
}

// PrefIntensityDetail は1都道府県の観測情報。
type PrefIntensityDetail struct {
	Name      string
	Code      string // 都道府県コード（例: "08"）
	MaxInt    string
	AreaNames []string // 観測地域名（最大5件）
}

// Key はS3/ファイル保存キーを返す（例: "eq/20260118034643/index.html"）。
func (d EarthquakeDetail) Key() string {
	return fmt.Sprintf("eq/%s/index.html", d.EventID)
}

// IntensityPillClass はMaxIntensityに対応するCSS pill classを返す。
func (d EarthquakeDetail) IntensityPillClass() string {
	return IntensityPillClassFor(d.MaxIntensity)
}

// IntensityPillClassFor は震度文字列からCSS pill classを返す。
func IntensityPillClassFor(s string) string {
	switch s {
	case "1":
		return "gray"
	case "2":
		return "s1"
	case "3", "4":
		return "s2"
	default: // 5-, 5+, 6-, 6+, 7
		return "s3"
	}
}

// MakeEarthquakeDetail は Report から EarthquakeDetail を生成する。
func MakeEarthquakeDetail(report Report) (EarthquakeDetail, bool) {
	if report.Body.Earthquake == nil {
		return EarthquakeDetail{}, false
	}

	eq := report.Body.Earthquake

	occurredAt := eq.OriginTime.Time
	if occurredAt.IsZero() {
		occurredAt = report.Head.ReportDateTime.Time
	}
	if occurredAt.IsZero() {
		return EarthquakeDetail{}, false
	}

	hypocenter := "不明"
	if eq.Hypocenter != nil && eq.Hypocenter.Area != nil {
		if n := strings.TrimSpace(eq.Hypocenter.Area.Name); n != "" {
			hypocenter = n
		}
	}

	var lat, lng float64
	var depth int
	coordinateKnown := false
	if eq.Hypocenter != nil && eq.Hypocenter.Area != nil {
		if coord := strings.TrimSpace(eq.Hypocenter.Area.Coordinate); coord != "" {
			if la, lo, d, ok := parseCoordinate(coord); ok {
				lat, lng, depth, coordinateKnown = la, lo, d, true
			}
		}
	}

	magnitude := 0.0
	magnitudeType := "M"
	if eq.Magnitude != nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(eq.Magnitude.Value), 64); err == nil {
			magnitude = v
		}
		if t := strings.TrimSpace(eq.Magnitude.Type); t != "" {
			magnitudeType = t
		}
	}

	maxIntensity := "-"
	if report.Body.Intensity != nil && report.Body.Intensity.Observation != nil {
		if v := strings.TrimSpace(report.Body.Intensity.Observation.MaxInt); v != "" {
			maxIntensity = v
		}
	}

	var prefs []PrefIntensityDetail
	if report.Body.Intensity != nil && report.Body.Intensity.Observation != nil {
		for _, p := range report.Body.Intensity.Observation.Prefs {
			pd := PrefIntensityDetail{
				Name:   p.Name,
				Code:   p.Code,
				MaxInt: p.MaxInt,
			}
			for i, a := range p.Areas {
				if i >= 5 {
					break
				}
				pd.AreaNames = append(pd.AreaNames, a.Name)
			}
			prefs = append(prefs, pd)
		}
	}
	sort.Slice(prefs, func(i, j int) bool {
		return prefIntensityOrder(prefs[i].MaxInt) > prefIntensityOrder(prefs[j].MaxInt)
	})

	return EarthquakeDetail{
		EventID:          strings.TrimSpace(report.Head.EventID),
		OccurredAt:       occurredAt,
		ReportedAt:       report.Head.ReportDateTime.Time,
		Status:           strings.TrimSpace(report.Control.Status),
		PublishingOffice: strings.TrimSpace(report.Control.PublishingOffice),
		HeadlineText:     strings.TrimSpace(report.Head.Headline.Text),
		Hypocenter:       hypocenter,
		Latitude:         lat,
		Longitude:        lng,
		Depth:            depth,
		CoordinateKnown:  coordinateKnown,
		Magnitude:        magnitude,
		MagnitudeType:    magnitudeType,
		MaxIntensity:     maxIntensity,
		ObservedPrefs:    prefs,
	}, true
}

// parseCoordinate はISO 6709形式の座標文字列をパースする。
// 例: "+36.0+137.4-10000/" → lat=36.0, lng=137.4, depthKm=10
func parseCoordinate(s string) (lat, lng float64, depthKm int, ok bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "/")
	re := regexp.MustCompile(`^([+-]\d+\.?\d*)([+-]\d+\.?\d*)([+-]\d+)$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, false
	}
	la, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, 0, 0, false
	}
	lo, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return 0, 0, 0, false
	}
	depthM, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return 0, 0, 0, false
	}
	// 深さはメートル単位、負値が地下を示す
	return la, lo, int(math.Round(math.Abs(depthM) / 1000)), true
}

// prefIntensityOrder は震度文字列を数値順序に変換する（ソート用）。
// summary.go の intensityOrder map と同じ定義だが関数として提供する。
func prefIntensityOrder(s string) int {
	if v, ok := intensityOrder[s]; ok {
		return v
	}
	return 0
}
