package domain

import "sort"

type RankingRecord struct {
	PrefName  string  `json:"pref"`
	Count     int     `json:"count"`
	Ratio     float64 `json:"ratio"`
	Intensity string  `json:"intensity"`
}

type RankingRecordList []RankingRecord

// 回数の降順にソートする
func (r RankingRecordList) SortByCountDesc() {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Count > r[j].Count
	})
}

// 震度の降順にソートする
func (r RankingRecordList) SortByIntensityDesc() {}

func ConvertReportListToRankingRecordList(reports ReportList) RankingRecordList {
	prefMap := MakePrefectureMap()
	var totalCount int

	for _, report := range reports {
		if report.Body.Intensity == nil {
			continue
		}
		// reportsひとつずつ見て、都道府県の登場回数をカウントする
		for _, pref := range report.Body.Intensity.Observation.Prefs {
			if data, exists := prefMap[pref.Code]; exists {
				data.Count++
				totalCount++
				// 都道府県ごとに最大震度を記録する
				if pref.MaxInt > data.Intensity {
					data.Intensity = pref.MaxInt
				}
				prefMap[pref.Code] = data
			}
		}
	}

	// 最終的にRankingRecordListを生成して返す
	var rankingList RankingRecordList
	for _, data := range prefMap {
		ratio := 0.0
		if totalCount > 0 {
			ratio = float64(data.Count) / float64(totalCount)
		}
		rankingList = append(rankingList, RankingRecord{
			PrefName:  data.Name,
			Count:     data.Count,
			Ratio:     ratio,
			Intensity: data.Intensity,
		})
	}
	return rankingList

}
