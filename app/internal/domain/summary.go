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
// weekScoreRanking は WeekScore 降順ソート済みであることを前提とする。
func BuildSummary(
	weekScoreRanking WeekScoreRecordList,
	weekPrefectureRanking RankingRecordList,
	surgeRanking SurgeRecordList,
	latestEarthquakes EarthquakeRecordList,
) string {
	jst := time.FixedZone("JST", 9*60*60)
	var parts []string

	// 1. 都道府県別地震スコア上位3都道府県を連記
	if len(weekScoreRanking) > 0 {
		top := weekScoreRanking[0]
		sentence := fmt.Sprintf(
			`今週の都道府県別地震スコア<a href="/score.html" title="地震スコアとは" style="margin-left:4px;display:inline-flex;align-items:center;justify-content:center;width:16px;height:16px;border-radius:999px;background:rgba(107,114,128,0.15);border:1px solid rgba(107,114,128,0.35);font-size:10px;font-weight:700;color:#6b7280;text-decoration:none;vertical-align:middle;line-height:1;">?</a>では%s（スコア%.1f・%d回）が最も高く`,
			top.PrefName, top.WeekScore, top.WeekCount,
		)
		if len(weekScoreRanking) >= 2 {
			r2 := weekScoreRanking[1]
			sentence += fmt.Sprintf("、次いで%s（%.1f）", r2.PrefName, r2.WeekScore)
		}
		if len(weekScoreRanking) >= 3 {
			r3 := weekScoreRanking[2]
			sentence += fmt.Sprintf("・%s（%.1f）", r3.PrefName, r3.WeekScore)
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
