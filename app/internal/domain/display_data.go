package domain

type DisplayData struct {
	RankingRecords         RankingRecordList    `json:"ranking_records"`
	LatestEarthquakes      EarthquakeRecordList `json:"latest_earthquakes"`
	TodayBigEarthquakes    EarthquakeRecordList `json:"today_big_earthquakes"`
	WeekBigEarthquakes     EarthquakeRecordList `json:"week_big_earthquakes"`
	WeekAllEarthquakes     EarthquakeRecordList `json:"week_all_earthquakes"`
	MonthBigEarthquakes    EarthquakeRecordList `json:"month_big_earthquakes"`
	TodayPrefectureRanking RankingRecordList    `json:"today_prefecture_ranking"`
	WeekPrefectureRanking  RankingRecordList    `json:"week_prefecture_ranking"`
	SurgeRanking           SurgeRecordList      `json:"surge_ranking"`
	WeekScoreRanking       WeekScoreRecordList  `json:"week_score_ranking"`
	WeekScoreRankingAll    WeekScoreRecordList  `json:"week_score_ranking_all"`
	HourlyEarthquake       HourlyEarthquakeData `json:"hourly_earthquake"`
	Summary                string               `json:"summary"`
	WeekReportCount        int                  `json:"week_report_count"`
}
