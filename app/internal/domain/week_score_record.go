package domain

import (
	"sort"
	"time"
)

// WeekScoreRecord は今週の都道府県別地震スコアを保持する。
type WeekScoreRecord struct {
	Rank       int     `json:"rank"`
	PrefCode   string  `json:"pref_code"`
	PrefName   string  `json:"pref_name"`
	WeekScore  float64 `json:"week_score"`  // G_long（τ=7日の震度重み付き減衰スコア）
	ShortScore float64 `json:"short_score"` // G_short（τ=1.5日の短期スコア）
	Ratio      float64 `json:"ratio"`       // 急上昇倍率 = G_short / (normalizedLong + ε)
	IsSurge    bool    `json:"is_surge"`    // 急上昇フラグ（minRatio を超えた場合 true）
	WeekCount  int     `json:"week_count"`  // 過去7日以内の地震件数
}

type WeekScoreRecordList []WeekScoreRecord

func (w WeekScoreRecordList) SortByScoreDesc() {
	sort.Slice(w, func(i, j int) bool {
		if w[i].WeekScore == w[j].WeekScore {
			if w[i].WeekCount == w[j].WeekCount {
				return w[i].PrefCode < w[j].PrefCode
			}
			return w[i].WeekCount > w[j].WeekCount
		}
		return w[i].WeekScore > w[j].WeekScore
	})
}

func (w WeekScoreRecordList) AssignRanks() {
	for i := range w {
		w[i].Rank = i + 1
	}
}

// MakeWeekScoreRecordList は過去1週間の震度重み付き減衰スコアで都道府県をランキングする。
// 急上昇フィルタは適用せず地震活動があった全都道府県を対象とするが、
// minRatio を満たす都道府県は IsSurge=true でマークする（minShortScore は使用しない）。
func MakeWeekScoreRecordList(
	allEarthquakes EarthquakeRecordList,
	now time.Time,
	minShortScore float64,
	minRatio float64,
	limit int,
) WeekScoreRecordList {
	if limit <= 0 {
		return WeekScoreRecordList{}
	}

	entries := buildPrefScores(allEarthquakes, now)
	list := make(WeekScoreRecordList, 0, len(entries))
	for _, e := range entries {
		isSurge := e.Ratio >= minRatio
		list = append(list, WeekScoreRecord{
			PrefCode:   e.PrefCode,
			PrefName:   e.PrefName,
			WeekScore:  e.LongScore,
			ShortScore: e.ShortScore,
			Ratio:      e.Ratio,
			IsSurge:    isSurge,
			WeekCount:  e.WeekCount,
		})
	}

	list.SortByScoreDesc()
	if len(list) > limit {
		list = list[:limit]
	}
	list.AssignRanks()

	return list
}
