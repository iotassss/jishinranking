package domain

import (
	"math"
	"sort"
	"time"
)

type SurgeRecord struct {
	Rank       int     `json:"rank"`
	PrefCode   string  `json:"pref_code"`
	PrefName   string  `json:"pref_name"`
	TodayCount int     `json:"today_count"`
	WeekAvg    float64 `json:"week_avg"`
	Score      float64 `json:"score"`
}

type SurgeRecordList []SurgeRecord

func (s SurgeRecordList) SortByScoreDesc() {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Score == s[j].Score {
			if s[i].TodayCount == s[j].TodayCount {
				return s[i].PrefCode < s[j].PrefCode
			}
			return s[i].TodayCount > s[j].TodayCount
		}
		return s[i].Score > s[j].Score
	})
}

func (s SurgeRecordList) AssignRanks() {
	for i := range s {
		s[i].Rank = i + 1
	}
}

// intensityWeight は JMA震度文字列から重みを返す。
// 重み: {1,2,4,8,16,24,40,64,100}（docs/ph2/急上昇計算.md 参照）
func intensityWeight(intensity string) float64 {
	switch intensity {
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 4
	case "4":
		return 8
	case "5-":
		return 16
	case "5+":
		return 24
	case "6-":
		return 40
	case "6+":
		return 64
	case "7":
		return 100
	default:
		return 0
	}
}

// surgeEvent は都道府県ごとの地震イベント（重み付き）。
type surgeEvent struct {
	OccurredAt time.Time
	Weight     float64
}

// calcDecayScore は時間減衰付きの地震スコアを計算する。
// tauDays: 時定数（日）— 大きいほど過去の地震も考慮する。
func calcDecayScore(events []surgeEvent, now time.Time, tauDays float64) float64 {
	var score float64
	for _, e := range events {
		days := now.Sub(e.OccurredAt).Hours() / 24.0
		if days < 0 {
			continue
		}
		score += e.Weight * math.Exp(-days/tauDays)
	}
	return score
}

// MakeSurgeRecordList は時間減衰スコアに基づく急上昇都道府県ランキングを生成する。
//
// アルゴリズム（docs/ph2/急上昇計算.md）:
//
//	短期スコア G_short = Σ w(I_i) * exp(-days/τ_short)  τ_short=1.5日（今日中心）
//	長期スコア G_long  = Σ w(I_i) * exp(-days/τ_long)   τ_long=7日（1週間ベースライン）
//	急上昇倍率 R       = G_short / (normalizedLong + ε)  normalizedLong = G_long * (τ_short/τ_long)
//	警戒スコア         = G_short * log(1 + R)  ← ランキング基準
//
// 判定条件: R > minRatio AND G_short > minShortScore
func MakeSurgeRecordList(
	allEarthquakes EarthquakeRecordList,
	now time.Time,
	minShortScore float64,
	minRatio float64,
	limit int,
) SurgeRecordList {
	if limit <= 0 {
		return SurgeRecordList{}
	}

	const (
		tauShort = 1.5 // 日（短期: 今日中心）
		tauLong  = 7.0 // 日（長期: 1週間ベースライン）
		epsilon  = 1.0 // ゼロ割防止
	)

	oneDayAgo := now.Add(-24 * time.Hour)
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)

	prefEvents := make(map[string][]surgeEvent)
	prefNames := make(map[string]string)
	prefTodayCount := make(map[string]int)
	prefWeekCount := make(map[string]int)

	for _, eq := range allEarthquakes {
		for _, pref := range eq.ObservedPrefs {
			if pref.Code == "" {
				continue
			}
			w := intensityWeight(pref.MaxInt)
			if w == 0 {
				continue
			}
			prefEvents[pref.Code] = append(prefEvents[pref.Code], surgeEvent{
				OccurredAt: eq.OccurredAt,
				Weight:     w,
			})
			if _, ok := prefNames[pref.Code]; !ok {
				prefNames[pref.Code] = pref.Name
			}
			if !eq.OccurredAt.Before(oneDayAgo) {
				prefTodayCount[pref.Code]++
			}
			if !eq.OccurredAt.Before(oneWeekAgo) {
				prefWeekCount[pref.Code]++
			}
		}
	}

	list := make(SurgeRecordList, 0)
	for prefCode, events := range prefEvents {
		short := calcDecayScore(events, now, tauShort)
		long := calcDecayScore(events, now, tauLong)
		// G_long を短期と同一時間スケールに正規化してから比較する。
		// 定常状態: short≈rate*τ_s, long≈rate*τ_l → normalizedLong≈rate*τ_s → ratio≈1
		// 急上昇時: short >> normalizedLong → ratio >> 1
		normalizedLong := long * (tauShort / tauLong)
		ratio := short / (normalizedLong + epsilon)

		if short < minShortScore || ratio < minRatio {
			continue
		}

		alertScore := short * math.Log1p(ratio)

		list = append(list, SurgeRecord{
			PrefCode:   prefCode,
			PrefName:   prefNames[prefCode],
			TodayCount: prefTodayCount[prefCode],
			WeekAvg:    float64(prefWeekCount[prefCode]) / 7.0,
			Score:      alertScore,
		})
	}

	list.SortByScoreDesc()
	if len(list) > limit {
		list = list[:limit]
	}
	list.AssignRanks()

	return list
}
