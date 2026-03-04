package domain

import (
	"sort"
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

func MakeSurgeRecordList(
	todayPrefMap map[string]PrefectureJishinData,
	weekPrefMap map[string]PrefectureJishinData,
	minTodayCount int,
	minScore float64,
	limit int,
) SurgeRecordList {
	if limit <= 0 {
		return SurgeRecordList{}
	}

	list := make(SurgeRecordList, 0, len(todayPrefMap))
	for prefCode, todayData := range todayPrefMap {
		weekData, exists := weekPrefMap[prefCode]
		if !exists {
			continue
		}

		todayCount := todayData.Count
		weekAvg := float64(weekData.Count) / 7.0
		score := (float64(todayCount) + 1.0) / (weekAvg + 1.0)

		if todayCount < minTodayCount {
			continue
		}
		if score < minScore {
			continue
		}

		list = append(list, SurgeRecord{
			PrefCode:   prefCode,
			PrefName:   todayData.Name,
			TodayCount: todayCount,
			WeekAvg:    weekAvg,
			Score:      score,
		})
	}

	list.SortByScoreDesc()
	if len(list) > limit {
		list = list[:limit]
	}
	list.AssignRanks()

	return list
}
