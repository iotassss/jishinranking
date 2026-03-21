package handler

// 原則依存するパッケージは標準ライブラリとdomainのみとする
import (
	"context"
	"log/slog"
	"time"

	"github.com/iotassss/jishinranking/internal/domain"
)

const htmlKey = "index.html"

// JMAからデータを取得する
type JMAFetcher interface {
	FetchEQVOLFeed(ctx context.Context, long bool) (domain.Feed, error)
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
	Save(ctx context.Context, key string, html domain.PublishedHTML) error
	Get(ctx context.Context, key string) (domain.PublishedHTML, error)
}

// 配信用HTMLを生成する
type HTMLGenerator interface {
	Generate(data domain.DisplayData, from, to, now time.Time) (domain.PublishedHTML, error)
	GenerateDetail(detail domain.EarthquakeDetail) (domain.PublishedHTML, error)
	GenerateAbout() (domain.PublishedHTML, error)
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
	FetchLongFeed bool,
) error {
	// 1週間分のデータを取得するための期間を設定
	now := time.Now()
	from := now.Add(-7 * 24 * time.Hour)
	to := now

	// JMAからフィードを取得する
	feed, err := h.jmaFetcher.FetchEQVOLFeed(ctx, FetchLongFeed)
	if err != nil {
		return err
	}

	// フィードから地震エントリーを抽出する
	earthquakeEntries := feed.FilterEarthquakeEntries()
	if len(earthquakeEntries) == 0 {
		// TODO: HTMLの更新日時と集計期間だけ更新する処理を入れる
		// if err := h.updateHTMLDatetime(ctx, htmlKey, now, from, to); err != nil {
		// 	return err
		// }
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
		// TODO: HTMLの更新日時と集計期間だけ更新する処理を入れる
		// if err := h.updateHTMLDatetime(ctx, htmlKey, now, from, to); err != nil {
		// 	return err
		// }
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
	latestEarthquakes := mergedReports.LatestEarthquakes(now, 6*time.Hour, 10)

	// 都道府県ごとの集計データを作成
	prefectureJishinDataMap := domain.MakePrefectureMapFromReportList(thisWeekReports)

	// HTML出力テーブル用に整形
	rankingRecords := domain.ConvertPrefectureJishinDataMapToRankingRecordList(prefectureJishinDataMap)
	rankingRecords.SortByCountDesc()
	rankingRecords.AssignRanks()
	rankingRecords.AssignTiers()

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

// func (h *Handler) updateHTMLDatetime(ctx context.Context, htmlKey string, now time.Time, from, to time.Time) error {
// 	publishedHTML, err := h.htmlRepo.Get(ctx, htmlKey)
// 	if err != nil {
// 		return err
// 	}
// 	updatedPublishedHTML := publishedHTML.UpdateUpdatedAt(now).UpdatePeriod(from, to)
// 	h.htmlRepo.Save(ctx, htmlKey, updatedPublishedHTML)

// 	return nil
// }
