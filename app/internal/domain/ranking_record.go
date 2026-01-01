package domain

import "sort"

type RankingRecord struct {
	Rank      int     `json:"rank,omitempty"`
	PrefCode  string  `json:"pref_code"`
	PrefName  string  `json:"pref"`
	Count     int     `json:"count"`
	Ratio     float64 `json:"ratio"`
	Intensity string  `json:"intensity"`
}

type RankingRecordList []RankingRecord

// 回数の降順にソートする
func (r RankingRecordList) SortByCountDesc() {
	sort.Slice(r, func(i, j int) bool {
		if r[i].Count == r[j].Count {
			return r[i].PrefCode < r[j].PrefCode
		}
		return r[i].Count > r[j].Count
	})
}

// 震度の降順にソートする
func (r RankingRecordList) SortByIntensityDesc() {}

func (r RankingRecordList) AssignRanks() {
	r.SortByCountDesc()
	currentRank := 1
	for i := 0; i < len(r); i++ {
		if i > 0 && r[i].Count < r[i-1].Count {
			currentRank = i + 1
		}
		r[i].Rank = currentRank
	}
}
