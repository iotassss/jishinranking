package domain

type DisplayData struct {
	RankingRecords    RankingRecordList    `json:"ranking_records"`
	LatestEarthquakes EarthquakeRecordList `json:"latest_earthquakes"`
}
