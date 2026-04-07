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
	IsSurge    bool    `json:"is_surge"`    // 急上昇フラグ（minShortScore かつ minRatio を超えた場合 true）
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
// minShortScore と minRatio を満たす都道府県は IsSurge=true でマークする。
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

	const (
		tauShort = 1.5 // 日（短期: 今日中心）
		tauLong  = 7.0 // 日（長期: 1週間ベースライン）
		epsilon  = 1.0 // ゼロ割防止
	)

	oneWeekAgo := now.Add(-7 * 24 * time.Hour)

	prefEvents := make(map[string][]surgeEvent)
	prefNames := make(map[string]string)
	prefWeekCount := make(map[string]int)

	for _, eq := range allEarthquakes {
		for _, pref := range eq.ObservedPrefs {
			if pref.Code == "" {
				continue
			}
			weight := intensityWeight(pref.MaxInt)
			if weight == 0 {
				continue
			}
			prefEvents[pref.Code] = append(prefEvents[pref.Code], surgeEvent{
				OccurredAt: eq.OccurredAt,
				Weight:     weight,
			})
			if _, ok := prefNames[pref.Code]; !ok {
				prefNames[pref.Code] = pref.Name
			}
			if !eq.OccurredAt.Before(oneWeekAgo) {
				prefWeekCount[pref.Code]++
			}
		}
	}

	list := make(WeekScoreRecordList, 0, len(prefEvents))
	for prefCode, events := range prefEvents {
		long := calcDecayScore(events, now, tauLong)
		short := calcDecayScore(events, now, tauShort)
		if long <= 0 {
			continue
		}
		normalizedLong := long * (tauShort / tauLong)
		ratio := short / (normalizedLong + epsilon)
		isSurge := short >= minShortScore && ratio >= minRatio
		list = append(list, WeekScoreRecord{
			PrefCode:   prefCode,
			PrefName:   prefNames[prefCode],
			WeekScore:  long,
			ShortScore: short,
			Ratio:      ratio,
			IsSurge:    isSurge,
			WeekCount:  prefWeekCount[prefCode],
		})
	}

	list.SortByScoreDesc()
	if len(list) > limit {
		list = list[:limit]
	}
	list.AssignRanks()

	return list
}
