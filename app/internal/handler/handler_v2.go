package handler

// 原則依存するパッケージは標準ライブラリとdomainのみとする
import (
	"context"
	"log/slog"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

func (h *Handler) ProcessV2(
	ctx context.Context,
	FetchLongFeed bool,
) error {
	// 1週間分のデータを取得するための期間を設定
	now := time.Now()
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)
	oneMonthAgo := now.Add(-30 * 24 * time.Hour)

	// ==============================
	// JMAデータ取得・保存処理
	// ==============================

	// JMAからフィードを取得する
	feed, err := h.jmaFetcher.FetchEQVOLFeed(ctx, FetchLongFeed)
	if err != nil {
		return err
	}

	// フィードから地震エントリーを抽出する
	earthquakeEntries := feed.FilterEarthquakeEntries()
	var reports domain.ReportList
	if len(earthquakeEntries) > 0 {
		// TODO: この処理はdomainに移す
		urls := make([]string, 0, len(earthquakeEntries))
		for _, entry := range earthquakeEntries {
			urls = append(urls, entry.Link.Href)
		}

		// JMAから最新の地震レポートを取得
		reports, err = h.jmaFetcher.FetchEarthquakeReport(ctx, urls)
		if err != nil {
			return err
		}
	}

	// 過去30日間の保存済みデータ取得
	oldReports, err := h.dataRepo.Get(ctx, &oneMonthAgo, &now)
	if err != nil {
		return err
	}
	oldReports = oldReports.FilterByTelegramCode(domain.TelegramCodeEarthquakeDetail)

	// 直近の保存済みレポートを取得
	latestOldReport := oldReports.Latest()
	// jmaから取得したreportsの中には過去のレポートも含まれるため、latestOldReport以降のものだけを抽出
	primaryReports := reports.NewerThan(latestOldReport)
	if len(primaryReports) > 0 {
		slog.Info("New earthquake reports found", "count", len(primaryReports))
		// 今回のレポートを保存
		if err := h.dataRepo.Save(ctx, primaryReports); err != nil {
			return err
		}
	}

	// ==============================
	// ランキング集計処理
	// ==============================

	// 既存データと今回のデータをmerge
	thisMonthReports := primaryReports.Merge(oldReports)

	// 7日前から現在時刻までのデータを抽出
	thisWeekReports := thisMonthReports.Between(oneWeekAgo, now)

	// 都道府県ごとの集計データを作成
	prefectureJishinDataMap := domain.MakePrefectureMapFromReportList(thisWeekReports)

	// HTML出力テーブル用に整形
	// ヒートマップのTierは回/1000km²（面積正規化）で算出する
	rankingRecords := domain.ConvertPrefectureJishinDataMapToAreaNormalizedRankingRecordList(prefectureJishinDataMap)
	rankingRecords.SortByCountDesc()
	rankingRecords.AssignRanks()
	rankingRecords.AssignTiers()

	// ==============================
	// 最新の地震
	// ==============================
	// 過去6時間以内の地震のうち大きい順に10件

	latestEarthquakes := thisMonthReports.LatestEarthquakes(now, 6*time.Hour, 10)

	// ==============================
	// 本日の大きい地震ランキング
	// ==============================
	// 過去24時間で発生したM3.0以上の地震のうち大きい順に10件

	todayBigEarthquakes := thisMonthReports.TopEarthquakesByMagnitude(now, 24*time.Hour, 3.0, 10)

	// ==============================
	// 今週の大きい地震ランキング
	// ==============================
	// 過去168時間で発生したM4.0以上の地震のうち大きい順に10件

	weekBigEarthquakes := thisMonthReports.TopEarthquakesByMagnitude(now, 168*time.Hour, 4.0, 10)

	// ==============================
	// 今月の大きい地震ランキング
	// ==============================
	// 過去720時間で発生したM5.0以上の地震のうち大きい順に10件

	monthBigEarthquakes := thisMonthReports.TopEarthquakesByMagnitude(now, 720*time.Hour, 5.0, 10)

	// ==============================
	// 本日の都道府県別地震回数ランキング
	// ==============================
	// 回/1000km²（47件）

	todayReports := thisMonthReports.Between(now.Add(-24*time.Hour), now)
	todayPrefectureMap := domain.MakePrefectureMapFromReportList(todayReports)
	todayPrefectureRanking := domain.ConvertPrefectureJishinDataMapToAreaNormalizedRankingRecordList(todayPrefectureMap)
	todayPrefectureRanking.SortByRatioDesc()
	todayPrefectureRanking.AssignRanksByRatio()
	todayPrefectureRanking.AssignTiers()

	// ==============================
	// 今週の都道府県別地震回数ランキング
	// ==============================
	// 回/1000km²（47件）

	// prefectureJishinDataMapはthisWeekReportsから生成済みのため再利用する
	weekPrefectureRanking := domain.ConvertPrefectureJishinDataMapToAreaNormalizedRankingRecordList(prefectureJishinDataMap)
	weekPrefectureRanking.SortByRatioDesc()
	weekPrefectureRanking.AssignRanksByRatio()
	weekPrefectureRanking.AssignTiers()

	// ==============================
	// 地震発生頻度急上昇都道府県（10件）
	// ==============================
	// score = (24h回数 + 1) / (7日平均 + 1)
	// 条件: (24h回数 ≥ 3) AND (score ≥ 2.0)

	surgeRanking := domain.MakeSurgeRecordList(todayPrefectureMap, prefectureJishinDataMap, 3, 2.0, 10)

	// ==============================
	// サマリー生成
	// ==============================
	// weekPrefectureRanking は回/1000km²降順ソート済みのため先頭が最高密度都道府県

	summary := domain.BuildSummary(weekPrefectureRanking, surgeRanking, latestEarthquakes)

	// ==============================
	// HTML生成・保存処理
	// ==============================

	displayData := domain.DisplayData{
		RankingRecords:         rankingRecords,
		LatestEarthquakes:      latestEarthquakes,
		TodayBigEarthquakes:    todayBigEarthquakes,
		WeekBigEarthquakes:     weekBigEarthquakes,
		MonthBigEarthquakes:    monthBigEarthquakes,
		TodayPrefectureRanking: todayPrefectureRanking,
		WeekPrefectureRanking:  weekPrefectureRanking,
		SurgeRanking:           surgeRanking,
		Summary:                summary,
	}

	// HTML生成・保存
	html, err := h.htmlGenerator.Generate(displayData, oneWeekAgo, now, now)
	if err != nil {
		return err
	}

	// // （ローカル）HTMLをファイルとしてここに保存する
	// filename := "index.html"
	// os.WriteFile(filename, []byte(html), 0o644)

	if err := h.htmlRepo.Save(ctx, htmlKey, html); err != nil {
		return err
	}

	return nil
}
