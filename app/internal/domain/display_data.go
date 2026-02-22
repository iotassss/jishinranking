package domain

type DisplayData struct {
	RankingRecords      RankingRecordList    `json:"ranking_records"`
	LatestEarthquakes   EarthquakeRecordList `json:"latest_earthquakes"`
	TodayBigEarthquakes EarthquakeRecordList `json:"today_big_earthquakes"`
	WeekBigEarthquakes  EarthquakeRecordList `json:"week_big_earthquakes"`
	MonthBigEarthquakes EarthquakeRecordList `json:"month_big_earthquakes"`
}
