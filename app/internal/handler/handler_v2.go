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
	// TODO: ここの日付を１ヶ月に延長することで、月間ランキングも集計できるようにする
	// 1週間分のデータを取得するための期間を設定
	now := time.Now()
	from := now.Add(-7 * 24 * time.Hour)
	to := now

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

	// 過去7日間の保存済みデータ取得
	oldReports, err := h.dataRepo.Get(ctx, &from, &to)
	if err != nil {
		return err
	}

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
	mergedReports := primaryReports.Merge(oldReports)

	// 7日前から現在時刻までのデータを抽出
	thisWeekReports := mergedReports.Between(from, to)

	// 都道府県ごとの集計データを作成
	prefectureJishinDataMap := domain.MakePrefectureMapFromReportList(thisWeekReports)

	// HTML出力テーブル用に整形
	rankingRecords := domain.ConvertPrefectureJishinDataMapToRankingRecordList(prefectureJishinDataMap)
	rankingRecords.SortByCountDesc()
	rankingRecords.AssignRanks()
	rankingRecords.AssignTiers()

	// ==============================
	// 最新の地震
	// ==============================
	// 過去6時間以内の地震のうち大きい順に10件
	latestEarthquakes := mergedReports.LatestEarthquakes(now, 6*time.Hour, 10)

	// ==============================
	// 本日の大きい地震ランキング
	// ==============================
	// 過去24時間で発生したM3.0以上の地震のうち大きい順に10件

	// ==============================
	// 今週の大きい地震ランキング
	// ==============================
	// 過去168時間で発生したM4.0以上の地震のうち大きい順に10件

	// ==============================
	// 今月の大きい地震ランキング
	// ==============================
	// 過去720時間で発生したM5.0以上の地震のうち大きい順に10件

	// ==============================
	// 本日の都道府県地震ランキング
	// ==============================
	// 回/1000km²（47件）

	// ==============================
	// 今週の都道府県地震ランキング
	// ==============================
	// 回/1000km²（47件）

	// ==============================
	// 地震発生頻度急上昇都道府県（10件）
	// ==============================
	// score = (24h回数 + 1) / (7日平均 + 1)
	// 条件: (24h回数 ≥ 3) AND (score ≥ 2.0)

	// ==============================
	// HTML生成・保存処理
	// ==============================

	displayData := domain.DisplayData{
		RankingRecords:    rankingRecords,
		LatestEarthquakes: latestEarthquakes,
	}

	// HTML生成・保存
	html, err := h.htmlGenerator.Generate(displayData, from, to, now)
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
