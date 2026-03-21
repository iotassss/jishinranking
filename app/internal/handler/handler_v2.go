package handler

// 原則依存するパッケージは標準ライブラリとdomainのみとする
import (
	"context"
	"fmt"
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

	// TODO: 以下のview向けデータ生計処理はview層に移すのが適切か要義論
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
	// 過去24時間で発生した最大震度2以上の地震のうち大きい順に10件

	todayBigEarthquakes := thisMonthReports.TopEarthquakesByMinIntensity(now, 24*time.Hour, 2, 10)

	// ==============================
	// 今週の大きい地震ランキング
	// ==============================
	// 過去168時間で発生した最大震度3以上の地震のうち大きい順に10件

	weekBigEarthquakes := thisMonthReports.TopEarthquakesByMinIntensity(now, 168*time.Hour, 3, 10)

	// ==============================
	// 今月の大きい地震ランキング
	// ==============================
	// 過去720時間で発生した最大震度4以上の地震のうち大きい順に10件

	monthBigEarthquakes := thisMonthReports.TopEarthquakesByMinIntensity(now, 720*time.Hour, 4, 10)

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
	// 1時間ごとの地震発生回数（過去1週間分）
	// ==============================
	// yyyy/mm/dd hh:00:00 ごと・都道府県ごとに回数を集計する
	// グラフ用途: 横軸 = 時間スロット、縦軸 = 発生回数

	hourlyEarthquake := domain.MakeHourlyEarthquakeData(thisWeekReports, now)

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
		HourlyEarthquake:       hourlyEarthquake,
		Summary:                summary,
		WeekReportCount:        len(thisWeekReports),
	}

	// HTML生成・保存
	html, err := h.htmlGenerator.Generate(displayData, oneWeekAgo, now, now)
	if err != nil {
		return err
	}

	if err := h.htmlRepo.Save(ctx, htmlKey, html); err != nil {
		return err
	}

	// ==============================
	// 詳細ページ生成・保存
	// ==============================
	// 過去30日分の VXSE53 レポートすべてについて毎回詳細HTMLを再生成する。
	// これにより、テンプレート変更が既存ページにも即時反映される。
	detailReports := thisMonthReports.FilterByTelegramCode(domain.TelegramCodeEarthquakeDetail)
	slog.Info("Generating detail pages", "count", len(detailReports))
	successCount := 0
	for _, report := range detailReports {
		detail, ok := domain.MakeEarthquakeDetail(report)
		if !ok {
			continue
		}
		detailHTML, err := h.htmlGenerator.GenerateDetail(detail)
		if err != nil {
			slog.Warn("detail HTML generation failed", "eventID", detail.EventID, "err", err)
			continue
		}
		if err := h.htmlRepo.Save(ctx, detail.Key(), detailHTML); err != nil {
			slog.Warn("detail HTML save failed", "eventID", detail.EventID, "err", err)
			continue
		}
		successCount++
	}
	slog.Info("Detail pages generated", "success", successCount, "total", len(detailReports))

	// ==============================
	// about ページ生成・保存
	// ==============================
	aboutHTML, err := h.htmlGenerator.GenerateAbout()
	if err != nil {
		return fmt.Errorf("about HTML generation failed: %w", err)
	}
	if err := h.htmlRepo.Save(ctx, "about.html", aboutHTML); err != nil {
		return fmt.Errorf("about HTML save failed: %w", err)
	}
	slog.Info("About page generated")

	return nil
}
