package handler

// 原則依存するパッケージは標準ライブラリとdomainのみとする
import (
	"context"
	"log/slog"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

// JMAからデータを取得する
type JMAFetcher interface {
	FetchEQVOLFeed(ctx context.Context) (domain.Feed, error)
	FetchEarthquakeReport(ctx context.Context, urls []string) (domain.ReportList, error)
}

// 地震データを保存・取得する
type DataRepo interface {
	// ファイル名は現在時刻（UTC）を元にしたキーで保存される（yyyymmddThhmmssZ.json）
	Save(ctx context.Context, reports domain.ReportList) error
	// from, toを指定してファイル名をフィルタリングして取得できる
	Get(ctx context.Context, from, to *time.Time) (domain.ReportList, error)
}

// 配信用HTMLを保存する
type HTMLRepo interface {
	Save(ctx context.Context, key string, html string) error
}

// 配信用HTMLを生成する
type HTMLGenerator interface {
	Generate(data domain.RankingRecordList) (string, error)
}

type Handler struct {
	jmaFetcher    JMAFetcher
	dataRepo      DataRepo
	htmlRepo      HTMLRepo
	htmlGenerator HTMLGenerator
}

func NewHandler(jmaFetcher JMAFetcher, dataRepo DataRepo, htmlRepo HTMLRepo, htmlGenerator HTMLGenerator) *Handler {
	return &Handler{
		jmaFetcher:    jmaFetcher,
		dataRepo:      dataRepo,
		htmlRepo:      htmlRepo,
		htmlGenerator: htmlGenerator,
	}
}

func (h *Handler) Process(
	ctx context.Context,
	dataKey,
	htmlKey string,
) error {
	// 1週間分のデータを取得するための期間を設定
	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	// JMAからフィードを取得する
	feed, err := h.jmaFetcher.FetchEQVOLFeed(ctx)
	if err != nil {
		return err
	}

	// フィードから地震エントリーを抽出する
	earthquakeEntries := feed.FilterEarthquakeEntries()
	if len(earthquakeEntries) == 0 {
		slog.Info("最新の地震エントリーがありません。処理を終了します。")
		return nil
	}

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
	if len(primaryReports) == 0 {
		slog.Info("新しい地震レポートがありません。処理を終了します。")
		return nil
	}

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
	tableRows := domain.ConvertPrefectureJishinDataMapToRankingRecordList(prefectureJishinDataMap)
	tableRows.SortByCountDesc()
	tableRows.AssignRanks()

	// HTML生成・保存
	html, err := h.htmlGenerator.Generate(tableRows)
	if err != nil {
		return err
	}
	if err := h.htmlRepo.Save(ctx, htmlKey, html); err != nil {
		return err
	}

	return nil
}
