package domain

// 都道府県コード（JIS X 0401）
const (
	Hokkaido  = "01"
	Aomori    = "02"
	Iwate     = "03"
	Miyagi    = "04"
	Akita     = "05"
	Yamagata  = "06"
	Fukushima = "07"
	Ibaraki   = "08"
	Tochigi   = "09"
	Gunma     = "10"
	Saitama   = "11"
	Chiba     = "12"
	Tokyo     = "13"
	Kanagawa  = "14"
	Niigata   = "15"
	Toyama    = "16"
	Ishikawa  = "17"
	Fukui     = "18"
	Yamanashi = "19"
	Nagano    = "20"
	Gifu      = "21"
	Shizuoka  = "22"
	Aichi     = "23"
	Mie       = "24"
	Shiga     = "25"
	Kyoto     = "26"
	Osaka     = "27"
	Hyogo     = "28"
	Nara      = "29"
	Wakayama  = "30"
	Tottori   = "31"
	Shimane   = "32"
	Okayama   = "33"
	Hiroshima = "34"
	Yamaguchi = "35"
	Tokushima = "36"
	Kagawa    = "37"
	Ehime     = "38"
	Kochi     = "39"
	Fukuoka   = "40"
	Saga      = "41"
	Nagasaki  = "42"
	Kumamoto  = "43"
	Oita      = "44"
	Miyazaki  = "45"
	Kagoshima = "46"
	Okinawa   = "47"
)

type PrefectureJishinData struct {
	Name      string `json:"name"`
	Count     int    `json:"count,omitempty"`
	Intensity string `json:"intensity,omitempty"`
}

// 編集せずコピーして使うこと
// Mapppは間違えてこのmapを書き換えてしまうのを防ぐための命名
var prefectureMappp = map[string]PrefectureJishinData{
	Hokkaido:  {Name: "北海道"},
	Aomori:    {Name: "青森県"},
	Iwate:     {Name: "岩手県"},
	Miyagi:    {Name: "宮城県"},
	Akita:     {Name: "秋田県"},
	Yamagata:  {Name: "山形県"},
	Fukushima: {Name: "福島県"},
	Ibaraki:   {Name: "茨城県"},
	Tochigi:   {Name: "栃木県"},
	Gunma:     {Name: "群馬県"},
	Saitama:   {Name: "埼玉県"},
	Chiba:     {Name: "千葉県"},
	Tokyo:     {Name: "東京都"},
	Kanagawa:  {Name: "神奈川県"},
	Niigata:   {Name: "新潟県"},
	Toyama:    {Name: "富山県"},
	Ishikawa:  {Name: "石川県"},
	Fukui:     {Name: "福井県"},
	Yamanashi: {Name: "山梨県"},
	Nagano:    {Name: "長野県"},
	Gifu:      {Name: "岐阜県"},
	Shizuoka:  {Name: "静岡県"},
	Aichi:     {Name: "愛知県"},
	Mie:       {Name: "三重県"},
	Shiga:     {Name: "滋賀県"},
	Kyoto:     {Name: "京都府"},
	Osaka:     {Name: "大阪府"},
	Hyogo:     {Name: "兵庫県"},
	Nara:      {Name: "奈良県"},
	Wakayama:  {Name: "和歌山県"},
	Tottori:   {Name: "鳥取県"},
	Shimane:   {Name: "島根県"},
	Okayama:   {Name: "岡山県"},
	Hiroshima: {Name: "広島県"},
	Yamaguchi: {Name: "山口県"},
	Tokushima: {Name: "徳島県"},
	Kagawa:    {Name: "香川県"},
	Ehime:     {Name: "愛媛県"},
	Kochi:     {Name: "高知県"},
	Fukuoka:   {Name: "福岡県"},
	Saga:      {Name: "佐賀県"},
	Nagasaki:  {Name: "長崎県"},
	Kumamoto:  {Name: "熊本県"},
	Oita:      {Name: "大分県"},
	Miyazaki:  {Name: "宮崎県"},
	Kagoshima: {Name: "鹿児島県"},
	Okinawa:   {Name: "沖縄県"},
}

func MakePrefectureMap() map[string]PrefectureJishinData {
	// prefectureMapppをコピーした新しいmapを作成して返す
	newMap := make(map[string]PrefectureJishinData)
	for k, v := range prefectureMappp {
		newMap[k] = v
	}
	return newMap
}

func MakePrefectureMapFromReportList(reports ReportList) map[string]PrefectureJishinData {
	prefMap := MakePrefectureMap()

	for _, report := range reports {
		if report.Body.Intensity == nil {
			continue
		}
		// reportsひとつずつ見て、都道府県の登場回数をカウントする
		for _, pref := range report.Body.Intensity.Observation.Prefs {
			if data, exists := prefMap[pref.Code]; exists {
				data.Count++
				// 都道府県ごとに最大震度を記録する
				if pref.MaxInt > data.Intensity {
					data.Intensity = pref.MaxInt
				}
				prefMap[pref.Code] = data
			}
		}
	}

	return prefMap
}

// ランキングは未設定のまま返す
func ConvertPrefectureJishinDataMapToRankingRecordList(prefMap map[string]PrefectureJishinData) RankingRecordList {
	totalCount := 0
	for _, data := range prefMap {
		totalCount += data.Count
	}
	var rankingList RankingRecordList
	for code, data := range prefMap {
		ratio := 0.0
		if totalCount > 0 {
			ratio = float64(data.Count) / float64(totalCount)
		}
		rankingList = append(rankingList, RankingRecord{
			Rank:      0, // ランキングは後で設定する
			PrefCode:  code,
			PrefName:  data.Name,
			Count:     data.Count,
			Ratio:     ratio,
			Intensity: data.Intensity,
		})
	}
	return rankingList
}
