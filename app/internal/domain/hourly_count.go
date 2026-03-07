package domain

import (
	"sort"
	"time"
)

// HourlyPrefectureCount は過去1週間における1時間単位・都道府県単位の地震発生回数を表す
type HourlyPrefectureCount struct {
	PrefCode string `json:"pref_code"`
	PrefName string `json:"pref_name"`
	// Counts[i] は Hours[i] の時間帯（HH:00:00 〜 HH:59:59）における発生回数
	Counts []int `json:"counts"`
}

// HourlyEarthquakeData は過去1週間の1時間ごと・都道府県ごとの地震発生回数データ
// グラフ用途: 横軸 = Hours、縦軸 = 各都道府県の Counts
type HourlyEarthquakeData struct {
	// Hours は過去168時間のスロット（古い順）。Hours[0] が 167 時間前、Hours[167] が直近の時間帯
	Hours []time.Time `json:"hours"`
	// Prefs は全 47 都道府県のデータ（都道府県コード順）
	Prefs []HourlyPrefectureCount `json:"prefs"`
}

const hourlySlots = 168 // 7日 × 24時間

// MakeHourlyEarthquakeData は now を基準に過去1週間の1時間ごと・都道府県ごとの地震発生回数を集計する
//
// スロット定義:
//
//	hours[0]   = now.Truncate(hour) - 167h  … 167時間前の時台
//	hours[167] = now.Truncate(hour)           … 直近の時台
//
// 各レポートは Body.Earthquake.OriginTime（発生時刻）を基準にスロットへ振り分ける
func MakeHourlyEarthquakeData(reports ReportList, now time.Time) HourlyEarthquakeData {
	nowHour := now.Truncate(time.Hour)

	// 168 時間スロットを生成（古い順）
	hours := make([]time.Time, hourlySlots)
	for i := 0; i < hourlySlots; i++ {
		hours[i] = nowHour.Add(time.Duration(i-(hourlySlots-1)) * time.Hour)
	}

	// 都道府県コード→カウント配列 を初期化（全 47 都道府県・0埋め）
	prefCountMap := make(map[string][]int, len(prefectureMappp))
	prefNameMap := make(map[string]string, len(prefectureMappp))
	for code, data := range prefectureMappp {
		prefCountMap[code] = make([]int, hourlySlots)
		prefNameMap[code] = data.Name
	}

	for _, report := range reports {
		if report.Body.Earthquake == nil || report.Body.Intensity == nil {
			continue
		}

		// 発生時刻を取得（OriginTime が空なら ReportDateTime にフォールバック）
		occurredAt := report.Body.Earthquake.OriginTime.Time
		if occurredAt.IsZero() {
			occurredAt = report.Head.ReportDateTime.Time
		}
		if occurredAt.IsZero() {
			continue
		}

		// スロットインデックスを算出
		reportHour := occurredAt.Truncate(time.Hour)
		slotIndex := int(reportHour.Sub(hours[0]).Hours())
		if slotIndex < 0 || slotIndex >= hourlySlots {
			continue
		}

		// 都道府県ごとにカウントを加算
		for _, pref := range report.Body.Intensity.Observation.Prefs {
			if counts, exists := prefCountMap[pref.Code]; exists {
				counts[slotIndex]++
				prefCountMap[pref.Code] = counts
			}
		}
	}

	// 都道府県コード順にスライスへ変換
	prefs := make([]HourlyPrefectureCount, 0, len(prefCountMap))
	for code, counts := range prefCountMap {
		prefs = append(prefs, HourlyPrefectureCount{
			PrefCode: code,
			PrefName: prefNameMap[code],
			Counts:   counts,
		})
	}
	sort.Slice(prefs, func(i, j int) bool {
		return prefs[i].PrefCode < prefs[j].PrefCode
	})

	return HourlyEarthquakeData{
		Hours: hours,
		Prefs: prefs,
	}
}
