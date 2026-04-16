package domain

import (
	"sort"
	"time"
)

// PrefPageData は都道府県別ページ生成用のデータ構造。
type PrefPageData struct {
	// 基本情報
	PrefCode string
	PrefSlug string // URLスラグ（例: "iwate"）
	PrefName string

	// 週間サマリー
	WeekRank      int     // 全国順位（密度ベース）
	WeekCount     int     // 今週の発生回数
	WeekRatio     float64 // 今週の密度（回/1000km²）
	WeekAvg       float64 // 7日平均（回/日）= WeekCount / 7
	WeekScoreRank int     // 全国順位（地震スコアベース）
	WeekScore     float64 // 今週の地震スコア（τ=7日）

	// 最大震度（今週）
	MaxIntensity   string    // "1"〜"7"（"-" = 不明）
	MaxIntensityAt time.Time // 最大震度を記録した地震の発生時刻

	// 本日（24h）
	TodayCount int // 今日（24h）の発生回数

	// 急上昇
	SurgeScore float64 // 急上昇スコア（0 = ランク外）
	SurgeRank  int     // 全国急上昇ランキング（0 = ランク外）

	// 全国比較
	NationalAvgRatio float64 // 全国平均密度（回/1000km²）

	// 最近の地震（この都道府県が観測された地震）
	RecentEarthquakes EarthquakeRecordList // 新しい順

	// 1時間ごとの発生回数（過去7日間・168スロット）
	HourlyHours  []time.Time // 時間スロット（古い順）
	HourlyCounts []int       // この都道府県のカウント（Hours に対応）

	// 震源地マップ用データ
	Epicenters []EpicenterPoint

	// ページメタ
	UpdatedAt time.Time
	WeekFrom  time.Time
	WeekTo    time.Time
}

// EpicenterPoint は震源地マップ用の1地震データ。
type EpicenterPoint struct {
	Lat       float64
	Lng       float64
	Magnitude float64
	// この都道府県での観測震度
	PrefMaxInt string
	Hypocenter string
	OccurredAt time.Time
	DetailURL  string
	EventID    string
}

// MakePrefPageData は全処理済みデータから特定都道府県のページデータを生成する。
//
// Parameters:
//
//	prefCode             - 対象都道府県コード（例: "04"）
//	weekPrefRanking      - 週間都道府県ランキング（密度降順・ランク付き済み）
//	todayPrefRanking     - 今日（24h）都道府県ランキング
//	surgeRanking         - 急上昇ランキング
//	weekReports          - 今週分のレポートリスト（ReportList）
//	hourlyData           - 全都道府県の時間別発生数
//	allWeekEarthquakes   - 今週全地震リスト（AllEarthquakesInPeriod 済み）
//	nationalAvgRatio     - 全国平均密度（事前計算済み）
//	weekScoreRanking     - 全都道府県の地震スコアランキング（AssignRanks済み）
//	now                  - 現在時刻
func MakePrefPageData(
	prefCode string,
	weekPrefRanking RankingRecordList,
	todayPrefRanking RankingRecordList,
	surgeRanking SurgeRecordList,
	hourlyData HourlyEarthquakeData,
	allWeekEarthquakes EarthquakeRecordList,
	nationalAvgRatio float64,
	weekScoreRanking WeekScoreRecordList,
	now, weekFrom, weekTo time.Time,
) PrefPageData {
	// --- 基本情報 ---
	prefName := prefectureMappp[prefCode].Name

	// --- 週間ランキング ---
	weekRank := 0
	weekCount := 0
	weekRatio := 0.0
	for _, r := range weekPrefRanking {
		if r.PrefCode == prefCode {
			weekRank = r.Rank
			weekCount = r.Count
			weekRatio = r.Ratio
			break
		}
	}

	// --- 本日（24h）カウント ---
	todayCount := 0
	for _, r := range todayPrefRanking {
		if r.PrefCode == prefCode {
			todayCount = r.Count
			break
		}
	}

	// --- 急上昇 ---
	surgeScore := 0.0
	surgeRank := 0
	for _, s := range surgeRanking {
		if s.PrefCode == prefCode {
			surgeScore = s.Score
			surgeRank = s.Rank
			break
		}
	}

	// --- 地震スコアランキング ---
	weekScoreRank := 0
	weekScore := 0.0
	for _, r := range weekScoreRanking {
		if r.PrefCode == prefCode {
			weekScoreRank = r.Rank
			weekScore = r.WeekScore
			break
		}
	}

	// --- 最近の地震（この都道府県が観測された地震） ---
	recentEarthquakes := make(EarthquakeRecordList, 0)
	for _, eq := range allWeekEarthquakes {
		for _, p := range eq.ObservedPrefs {
			if p.Code == prefCode {
				recentEarthquakes = append(recentEarthquakes, eq)
				break
			}
		}
	}
	// 新しい順に並べ直す（allWeekEarthquakes はすでに新しい順だが念のため）
	sort.Slice(recentEarthquakes, func(i, j int) bool {
		return recentEarthquakes[i].OccurredAt.After(recentEarthquakes[j].OccurredAt)
	})

	// --- 最大震度（今週）---
	maxIntensity := "-"
	var maxIntensityAt time.Time
	for _, eq := range recentEarthquakes {
		for _, p := range eq.ObservedPrefs {
			if p.Code == prefCode {
				if IntensityLevel(p.MaxInt) > IntensityLevel(maxIntensity) {
					maxIntensity = p.MaxInt
					maxIntensityAt = eq.OccurredAt
				}
				break
			}
		}
	}

	// --- 1時間ごとカウント（この都道府県のみ） ---
	hourlyCounts := make([]int, len(hourlyData.Hours))
	for _, pc := range hourlyData.Prefs {
		if pc.PrefCode == prefCode {
			copy(hourlyCounts, pc.Counts)
			break
		}
	}

	// --- 震源地マップ用データ ---
	epicenters := make([]EpicenterPoint, 0, len(recentEarthquakes))
	for _, eq := range recentEarthquakes {
		prefMaxInt := "-"
		for _, p := range eq.ObservedPrefs {
			if p.Code == prefCode {
				prefMaxInt = p.MaxInt
				break
			}
		}
		epicenters = append(epicenters, EpicenterPoint{
			Lat:        eq.Latitude,
			Lng:        eq.Longitude,
			Magnitude:  eq.Magnitude,
			PrefMaxInt: prefMaxInt,
			Hypocenter: eq.Hypocenter,
			OccurredAt: eq.OccurredAt,
			DetailURL:  "/eq/" + eq.EventID + "/",
			EventID:    eq.EventID,
		})
	}

	weekAvg := float64(weekCount) / 7.0

	return PrefPageData{
		PrefCode:          prefCode,
		PrefSlug:          PrefSlug(prefCode),
		PrefName:          prefName,
		WeekRank:          weekRank,
		WeekCount:         weekCount,
		WeekRatio:         weekRatio,
		WeekAvg:           weekAvg,
		WeekScoreRank:     weekScoreRank,
		WeekScore:         weekScore,
		MaxIntensity:      maxIntensity,
		MaxIntensityAt:    maxIntensityAt,
		TodayCount:        todayCount,
		SurgeScore:        surgeScore,
		SurgeRank:         surgeRank,
		NationalAvgRatio:  nationalAvgRatio,
		RecentEarthquakes: recentEarthquakes,
		HourlyHours:       hourlyData.Hours,
		HourlyCounts:      hourlyCounts,
		Epicenters:        epicenters,
		UpdatedAt:         now,
		WeekFrom:          weekFrom,
		WeekTo:            weekTo,
	}
}

// CalcNationalAvgRatio は全都道府県の平均密度（回/1000km²）を計算する。
func CalcNationalAvgRatio(weekPrefRanking RankingRecordList) float64 {
	if len(weekPrefRanking) == 0 {
		return 0
	}
	total := 0.0
	count := 0
	for _, r := range weekPrefRanking {
		total += r.Ratio
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// PrefCodeList は全47都道府県コードを順番に返す。
var PrefCodeList = []string{
	Hokkaido, Aomori, Iwate, Miyagi, Akita, Yamagata, Fukushima,
	Ibaraki, Tochigi, Gunma, Saitama, Chiba, Tokyo, Kanagawa,
	Niigata, Toyama, Ishikawa, Fukui, Yamanashi, Nagano,
	Gifu, Shizuoka, Aichi, Mie, Shiga, Kyoto, Osaka, Hyogo, Nara, Wakayama,
	Tottori, Shimane, Okayama, Hiroshima, Yamaguchi,
	Tokushima, Kagawa, Ehime, Kochi,
	Fukuoka, Saga, Nagasaki, Kumamoto, Oita, Miyazaki, Kagoshima, Okinawa,
}
