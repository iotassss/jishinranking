package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type EarthquakeRecord struct {
	EventID       string                `json:"event_id"`
	OccurredAt    time.Time             `json:"occurred_at"`
	Hypocenter    string                `json:"hypocenter"`
	Magnitude     float64               `json:"magnitude"`
	MaxIntensity  string                `json:"max_intensity"`
	DetailURL     string                `json:"detail_url"`
	ObservedPrefs []PrefIntensityDetail `json:"observed_prefs"`
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

// intensityToInt は最大震度の文字列を比較用の整数に変換する。
// 大きいほど強い震度を表す。
func intensityToInt(s string) int {
	switch s {
	case "7":
		return 9
	case "6強":
		return 8
	case "6弱":
		return 7
	case "5強":
		return 6
	case "5弱":
		return 5
	case "4":
		return 4
	case "3":
		return 3
	case "2":
		return 2
	case "1":
		return 1
	default:
		return 0
	}
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

	return EarthquakeRecord{
		EventID:       strings.TrimSpace(report.Head.EventID),
		OccurredAt:    occurredAt,
		Hypocenter:    hypocenter,
		Magnitude:     magnitude,
		MaxIntensity:  maxIntensity,
		DetailURL:     report.Metadata.URL,
		ObservedPrefs: prefs,
	}, true
}
