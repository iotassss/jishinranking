package handler

// 原則依存するパッケージは標準ライブラリとdomainのみとする
import (
	"context"
	"os"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

type InitializeHandler struct {
	jmaFetcher    JMAFetcher
	dataRepo      DataRepo
	htmlRepo      HTMLRepo
	htmlGenerator HTMLGenerator
}

func NewInitializeHandler(jmaFetcher JMAFetcher, dataRepo DataRepo, htmlRepo HTMLRepo, htmlGenerator HTMLGenerator) *InitializeHandler {
	return &InitializeHandler{
		jmaFetcher:    jmaFetcher,
		dataRepo:      dataRepo,
		htmlRepo:      htmlRepo,
		htmlGenerator: htmlGenerator,
	}
}

func (h *InitializeHandler) Process(
	ctx context.Context,
	dataKey,
	htmlKey string,
) error {
	// 1週間分のデータを取得するための期間を設定
	now := time.Now()
	from := now.Add(-7 * 24 * time.Hour)
	to := now

	// JMAからフィードを取得する
	feed, err := h.jmaFetcher.FetchEQVOLFeed(ctx, true)
	if err != nil {
		return err
	}

	// フィードから地震エントリーを抽出する
	earthquakeEntries := feed.FilterEarthquakeEntries()

	// TODO: この処理はdomainに移す
	urls := make([]string, 0, len(earthquakeEntries))
	for _, entry := range earthquakeEntries {
		urls = append(urls, entry.Link.Href)
	}

	// JMAから最新の地震レポートを取得
	reports, err := h.jmaFetcher.FetchEarthquakeReport(ctx, urls)
	if err != nil {
		return err
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

	// 今回のレポートを保存
	if err := h.dataRepo.Save(ctx, primaryReports); err != nil {
		return err
	}

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

	// HTML生成・保存
	html, err := h.htmlGenerator.Generate(rankingRecords, from, to, now)
	if err != nil {
		return err
	}

	// HTMLをファイルとしてここに保存する
	filename := "index.html"
	os.WriteFile(filename, []byte(html), 0o644)

	if err := h.htmlRepo.Save(ctx, htmlKey, domain.PublishedHTML(html)); err != nil {
		return err
	}

	return nil
}
