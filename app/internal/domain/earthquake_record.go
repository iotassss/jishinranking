package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type EarthquakeRecord struct {
	EventID         string                `json:"event_id"`
	OccurredAt      time.Time             `json:"occurred_at"`
	Hypocenter      string                `json:"hypocenter"`
	Latitude        float64               `json:"latitude"`
	Longitude       float64               `json:"longitude"`
	CoordinateKnown bool                  `json:"coordinate_known"`
	Magnitude       float64               `json:"magnitude"`
	MaxIntensity    string                `json:"max_intensity"`
	DetailURL       string                `json:"detail_url"`
	ObservedPrefs   []PrefIntensityDetail `json:"observed_prefs"`
}

type EarthquakeRecordList []EarthquakeRecord

func (rl ReportList) LatestEarthquakes(now time.Time, within time.Duration, limit int) EarthquakeRecordList {
	return rl.TopEarthquakesByMagnitude(now, within, 0.0, limit)
}

func (rl ReportList) TopEarthquakesByMagnitude(now time.Time, within time.Duration, minMagnitude float64, limit int) EarthquakeRecordList {
	if limit <= 0 {
		return EarthquakeRecordList{}
	}

	from := now.Add(-within)
	records := make(EarthquakeRecordList, 0, len(rl))
	for _, report := range rl {
		record, ok := makeEarthquakeRecord(report)
		if !ok {
			continue
		}
		if record.OccurredAt.Before(from) || record.OccurredAt.After(now) {
			continue
		}
		if record.Magnitude < minMagnitude {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		ii := intensityToInt(records[i].MaxIntensity)
		ij := intensityToInt(records[j].MaxIntensity)
		if ii != ij {
			return ii > ij
		}
		if records[i].Magnitude != records[j].Magnitude {
			return records[i].Magnitude > records[j].Magnitude
		}
		return records[i].OccurredAt.After(records[j].OccurredAt)
	})

	if len(records) > limit {
		return records[:limit]
	}
	return records
}

// TopEarthquakesByMinIntensity は指定期間内で最大震度が minIntensityInt 以上の地震を
// 震度降順・マグニチュード降順で最大 limit 件返す。
// minIntensityInt は intensityToInt の返り値と同じスケール（例: 震度2→2, 震度3→3, 震度4→4）。
func (rl ReportList) TopEarthquakesByMinIntensity(now time.Time, within time.Duration, minIntensityInt int, limit int) EarthquakeRecordList {
	if limit <= 0 {
		return EarthquakeRecordList{}
	}

	from := now.Add(-within)
	records := make(EarthquakeRecordList, 0, len(rl))
	for _, report := range rl {
		record, ok := makeEarthquakeRecord(report)
		if !ok {
			continue
		}
		if record.OccurredAt.Before(from) || record.OccurredAt.After(now) {
			continue
		}
		if intensityToInt(record.MaxIntensity) < minIntensityInt {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		ii := intensityToInt(records[i].MaxIntensity)
		ij := intensityToInt(records[j].MaxIntensity)
		if ii != ij {
			return ii > ij
		}
		if records[i].Magnitude != records[j].Magnitude {
			return records[i].Magnitude > records[j].Magnitude
		}
		return records[i].OccurredAt.After(records[j].OccurredAt)
	})

	if len(records) > limit {
		return records[:limit]
	}
	return records
}

// AllEarthquakesInPeriod は指定期間内のすべての地震を発生時刻の降順で返す。
// 同一 EventID の重複は除外し、ObservedPrefs が多い（より詳細な）記録を採用する。
func (rl ReportList) AllEarthquakesInPeriod(now time.Time, within time.Duration) EarthquakeRecordList {
	from := now.Add(-within)
	seen := make(map[string]EarthquakeRecord)
	for _, report := range rl {
		record, ok := makeEarthquakeRecord(report)
		if !ok {
			continue
		}
		if record.OccurredAt.Before(from) || record.OccurredAt.After(now) {
			continue
		}
		prev, exists := seen[record.EventID]
		if !exists || len(record.ObservedPrefs) > len(prev.ObservedPrefs) {
			seen[record.EventID] = record
		}
	}
	records := make(EarthquakeRecordList, 0, len(seen))
	for _, r := range seen {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].OccurredAt.After(records[j].OccurredAt)
	})
	return records
}

// IntensityLevel は最大震度の文字列を比較用の整数（1〜9）に変換する。公開版。
func IntensityLevel(s string) int {
	return intensityToInt(s)
}

// IntensityDisplay は震度文字列を表示用の日本語形式に変換する。
// JMA XML の "5-"→"5弱", "5+"→"5強", "6-"→"6弱", "6+"→"6強"。
// それ以外の値はそのまま返す。
func IntensityDisplay(s string) string {
	switch s {
	case "5-":
		return "5弱"
	case "5+":
		return "5強"
	case "6-":
		return "6弱"
	case "6+":
		return "6強"
	default:
		return s
	}
}

// intensityToInt は最大震度の文字列を比較用の整数に変換する。
// JMA XML の表記（"5-"=震度5弱, "5+"=震度5強, "6-"=震度6弱, "6+"=震度6強）に対応する。
// 大きいほど強い震度を表す。
func intensityToInt(s string) int {
	if v, ok := intensityOrder[s]; ok {
		return v
	}
	return 0
}

func makeEarthquakeRecord(report Report) (EarthquakeRecord, bool) {
	if report.Body.Earthquake == nil {
		return EarthquakeRecord{}, false
	}

	occurredAt := report.Body.Earthquake.OriginTime.Time
	if occurredAt.IsZero() {
		occurredAt = report.Head.ReportDateTime.Time
	}
	if occurredAt.IsZero() {
		return EarthquakeRecord{}, false
	}

	hypocenter := "不明"
	if report.Body.Earthquake.Hypocenter != nil && report.Body.Earthquake.Hypocenter.Area != nil {
		name := strings.TrimSpace(report.Body.Earthquake.Hypocenter.Area.Name)
		if name != "" {
			hypocenter = name
		}
	}

	maxIntensity := "-"
	if report.Body.Intensity != nil && report.Body.Intensity.Observation != nil {
		v := strings.TrimSpace(report.Body.Intensity.Observation.MaxInt)
		if v != "" {
			maxIntensity = v
		}
	}

	magnitude := 0.0
	if report.Body.Earthquake.Magnitude != nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(report.Body.Earthquake.Magnitude.Value), 64); err == nil {
			magnitude = v
		}
	}

	var prefs []PrefIntensityDetail
	if report.Body.Intensity != nil && report.Body.Intensity.Observation != nil {
		for _, p := range report.Body.Intensity.Observation.Prefs {
			prefs = append(prefs, PrefIntensityDetail{
				Name:   p.Name,
				Code:   p.Code,
				MaxInt: p.MaxInt,
			})
		}
		sort.Slice(prefs, func(i, j int) bool {
			return prefIntensityOrder(prefs[i].MaxInt) > prefIntensityOrder(prefs[j].MaxInt)
		})
	}

	var lat, lng float64
	coordinateKnown := false
	if report.Body.Earthquake.Hypocenter != nil && report.Body.Earthquake.Hypocenter.Area != nil {
		if coord := strings.TrimSpace(report.Body.Earthquake.Hypocenter.Area.Coordinate); coord != "" {
			if la, lo, _, ok := parseCoordinate(coord); ok {
				lat, lng, coordinateKnown = la, lo, true
			}
		}
	}

	eventID := strings.TrimSpace(report.Head.EventID)
	return EarthquakeRecord{
		EventID:         eventID,
		OccurredAt:      occurredAt,
		Hypocenter:      hypocenter,
		Latitude:        lat,
		Longitude:       lng,
		CoordinateKnown: coordinateKnown,
		Magnitude:       magnitude,
		MaxIntensity:    maxIntensity,
		DetailURL:       "/eq/" + eventID + "/index.html",
		ObservedPrefs:   prefs,
	}, true
}
