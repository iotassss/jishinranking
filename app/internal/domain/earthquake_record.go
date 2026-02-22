package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type EarthquakeRecord struct {
	OccurredAt   time.Time `json:"occurred_at"`
	Hypocenter   string    `json:"hypocenter"`
	Magnitude    float64   `json:"magnitude"`
	MaxIntensity string    `json:"max_intensity"`
	DetailURL    string    `json:"detail_url"`
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
		if records[i].Magnitude == records[j].Magnitude {
			return records[i].OccurredAt.After(records[j].OccurredAt)
		}
		return records[i].Magnitude > records[j].Magnitude
	})

	if len(records) > limit {
		return records[:limit]
	}
	return records
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

	return EarthquakeRecord{
		OccurredAt:   occurredAt,
		Hypocenter:   hypocenter,
		Magnitude:    magnitude,
		MaxIntensity: maxIntensity,
		DetailURL:    report.Metadata.URL,
	}, true
}
