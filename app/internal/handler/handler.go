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
	FetchEarthquakeReport(ctx context.Context) (domain.ReportList, error)
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
	reports, err := h.jmaFetcher.FetchEarthquakeReport(ctx)
	if err != nil {
		return err
	}
	if len(reports) == 0 {
		slog.Info("最新の地震レポートがありません。処理を終了します。")
		return nil
	}

	// 過去7日間の保存済みデータ取得
	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()
	oldReports, err := h.dataRepo.Get(ctx, &from, &to)
	if err != nil {
		return err
	}

	// 直近の保存済みレポートを取得
	latestOldReport := oldReports.Latest()
	// jmaから取得したreportsの中には過去のレポートも含まれるため、latestOldReport以降のものだけを抽出
	primaryReports := reports.After(latestOldReport)

	// 今回のレポートを保存
	if err := h.dataRepo.Save(ctx, primaryReports); err != nil {
		return err
	}

	// HTML生成・保存
	mergedReports := primaryReports.Merge(oldReports)
	rankingRecordList := domain.ConvertReportListToRankingRecordList(mergedReports)
	html, err := h.htmlGenerator.Generate(rankingRecordList)
	if err != nil {
		return err
	}
	if err := h.htmlRepo.Save(ctx, htmlKey, html); err != nil {
		return err
	}

	return nil
}
