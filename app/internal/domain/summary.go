package domain

import (
	"fmt"
	"strings"
	"time"
)

var intensityOrder = map[string]int{
	"-":  0,
	"1":  1,
	"2":  2,
	"3":  3,
	"4":  4,
	"5-": 5,
	"5+": 6,
	"6-": 7,
	"6+": 8,
	"7":  9,
}

func intensityAtLeast(intensity, min string) bool {
	iv, ok1 := intensityOrder[intensity]
	mv, ok2 := intensityOrder[min]
	if !ok1 || !ok2 {
		return false
	}
	return iv >= mv
}

// BuildSummary は集計データから約200文字の日本語サマリーを生成する。
// weekPrefectureRanking は回/1000km²の降順ソート済みであることを前提とする。
func BuildSummary(
	weekPrefectureRanking RankingRecordList,
	surgeRanking SurgeRecordList,
	latestEarthquakes EarthquakeRecordList,
) string {
	jst := time.FixedZone("JST", 9*60*60)
	var parts []string

	// 1. 地震の多さ（回/1000km²）トップ3都道府県を連記
	activeRanking := make(RankingRecordList, 0, len(weekPrefectureRanking))
	for _, r := range weekPrefectureRanking {
		if r.Ratio > 0 {
			activeRanking = append(activeRanking, r)
		}
	}
	if len(activeRanking) > 0 {
		top := activeRanking[0]
		sentence := fmt.Sprintf(
			"面積あたりの地震の多さ（回/1000km²）で見ると%s（%.1f回・%d回）が首位",
			top.PrefName, top.Ratio, top.Count,
		)
		if len(activeRanking) >= 2 {
			r2 := activeRanking[1]
			sentence += fmt.Sprintf("、次いで%s（%.1f回）", r2.PrefName, r2.Ratio)
		}
		if len(activeRanking) >= 3 {
			r3 := activeRanking[2]
			sentence += fmt.Sprintf("・%s（%.1f回）", r3.PrefName, r3.Ratio)
		}
		parts = append(parts, sentence+"と続きます。")
	}

	// 2. 震度3以上を観測した都道府県数と最大震度の都道府県
	intensity3Count := 0
	maxIntensityPref := ""
	maxIntensityVal := ""
	for _, r := range weekPrefectureRanking {
		if intensityAtLeast(r.Intensity, "3") {
			intensity3Count++
		}
		if r.Intensity > maxIntensityVal {
			maxIntensityVal = r.Intensity
			maxIntensityPref = r.PrefName
		}
	}
	if intensity3Count > 0 && maxIntensityPref != "" {
		parts = append(parts, fmt.Sprintf(
			"震度3以上を観測した都道府県は%d件で、最大震度%sは%sで記録されました。",
			intensity3Count, IntensityDisplay(maxIntensityVal), maxIntensityPref,
		))
	} else if intensity3Count == 0 && len(weekPrefectureRanking) > 0 {
		parts = append(parts, "集計期間中に震度3以上の揺れは観測されませんでした。")
	}

	// 3. 急上昇都道府県（1位のみ）
	if len(surgeRanking) > 0 {
		s := surgeRanking[0]
		parts = append(parts, fmt.Sprintf(
			"地震発生頻度が急増している地域では%sが直近24時間に%d回（週平均比約%.1f倍）と顕著な増加傾向を示しています。",
			s.PrefName, s.TodayCount, s.Score,
		))
	}

	// 4. 直近の地震（最大震度のもの）
	if len(latestEarthquakes) > 0 {
		eq := latestEarthquakes[0]
		parts = append(parts, fmt.Sprintf(
			"直近6時間では%sを震源とする震度%s・M%.1fの地震が観測されています（%s）。",
			eq.Hypocenter, IntensityDisplay(eq.MaxIntensity), eq.Magnitude, eq.OccurredAt.In(jst).Format("01/02 15:04"),
		))
	}

	if len(parts) == 0 {
		return "現在、集計対象期間内の地震データはありません。"
	}
	return strings.Join(parts, "")
}
