package domain

import (
	"sort"
	"strconv"
)

type RankingRecord struct {
	Rank      int     `json:"rank,omitempty"`
	PrefCode  string  `json:"pref_code"`
	PrefName  string  `json:"pref"`
	Count     int     `json:"count"`
	Ratio     float64 `json:"ratio"`
	Intensity string  `json:"intensity"`
	Tier      int     `json:"tier,omitempty"`
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

// ratioが0.0のRankingRecordはtierを0にする
// それ以外は以下のルールでtierを設定する
// ratioが1種類の時は全てに3を設定する
// ratioが2種類の時は3,5を設定する
// ratioが3種類の時は1,3,5を設定する
// ratioが4種類の時は1,2,3,5を設定する
// ratioが5種類の時は1,2,3,4,5を設定する
// ここからは1つのtierに複数のratioが割り当てられる場合がある
// ratioが6種類の時は1(2種類),2,3,4,5を設定する
// ratioが7種類の時は1(2種類),2(2種類),3,4,5を設定する
// ratioが8種類の時は1(2種類),2(2種類),3(2種類),4,5を設定する
// ratioが9種類の時は1(2種類),2(2種類),3(2種類),4(2種類),5を設定する
// ratioが10種類の時は1(3種類),2(2種類),3(2種類),4(2種類),5を設定する
// ratioが11種類の時は1(3種類),2(3種類),3(2種類),4(2種類),5を設定する
// ratioが12種類の時は1(3種類),2(3種類),3(3種類),4(2種類),5を設定する
// ...
func (r RankingRecordList) AssignTiers() {
	const zero = 0.0

	// ratio のユニーク集合（0除外）
	keyOf := func(x float64) string {
		return strconv.FormatFloat(x, 'g', -1, 64) // ratioは事前に丸め済み想定
	}

	uniq := make(map[string]float64, len(r))
	for i := range r {
		if r[i].Ratio == zero {
			continue
		}
		uniq[keyOf(r[i].Ratio)] = r[i].Ratio
	}

	// ユニーク ratio を昇順に並べる
	ratios := make([]float64, 0, len(uniq))
	for _, v := range uniq {
		ratios = append(ratios, v)
	}
	sort.Float64s(ratios)

	k := len(ratios)
	if k == 0 {
		for i := range r {
			r[i].Tier = 0
		}
		return
	}

	// ratio(昇順) -> tier
	tierByKey := make(map[string]int, k)
	assignByBuckets := func(bucketSizes [5]int) {
		idx := 0
		for tier := 1; tier <= 5; tier++ {
			for c := 0; c < bucketSizes[tier-1] && idx < k; c++ {
				tierByKey[keyOf(ratios[idx])] = tier
				idx++
			}
		}
	}

	switch k {
	case 1:
		tierByKey[keyOf(ratios[0])] = 3
	case 2:
		tierByKey[keyOf(ratios[0])] = 3
		tierByKey[keyOf(ratios[1])] = 5
	case 3:
		tierByKey[keyOf(ratios[0])] = 1
		tierByKey[keyOf(ratios[1])] = 3
		tierByKey[keyOf(ratios[2])] = 5
	case 4:
		tierByKey[keyOf(ratios[0])] = 1
		tierByKey[keyOf(ratios[1])] = 2
		tierByKey[keyOf(ratios[2])] = 3
		tierByKey[keyOf(ratios[3])] = 5
	case 5:
		for i := 0; i < 5; i++ {
			tierByKey[keyOf(ratios[i])] = i + 1
		}
	case 6:
		assignByBuckets([5]int{2, 1, 1, 1, 1})
	case 7:
		assignByBuckets([5]int{2, 2, 1, 1, 1})
	case 8:
		// 変更後: 1(2),2(2),3(2),4,5
		assignByBuckets([5]int{2, 2, 2, 1, 1})
	case 9:
		assignByBuckets([5]int{2, 2, 2, 2, 1})
	default:
		// k>=10: 以降は「1から順に3個まで増やしつつ」埋めていく（例: 10=3,2,2,2,1 / 11=3,3,2,2,1 / 12=3,3,3,2,1 ...）
		// 基本形: [2,2,2,2,1] (k=9) から、余りを tier1 -> tier2 -> tier3 -> tier4 の順に加算（各tier最大+1=3個まで）
		b := [5]int{2, 2, 2, 2, 1}
		need := k - 9
		for need > 0 {
			for i := 0; i < 4 && need > 0; i++ { // tier1..tier4
				if b[i] < 3 {
					b[i]++
					need--
				}
			}
		}
		assignByBuckets(b)
	}

	// レコードへ反映（0は tier0）
	for i := range r {
		if r[i].Ratio == zero {
			r[i].Tier = 0
			continue
		}
		r[i].Tier = tierByKey[keyOf(r[i].Ratio)]
	}
}
