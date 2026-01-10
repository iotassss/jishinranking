package main

import (
	"log/slog"

	"github.com/iotassss/jishinranking/internal/domain"
)

func main() {
	rankingRecordList := domain.RankingRecordList{
		{PrefName: "PrefA", Ratio: 0.0},
		{PrefName: "PrefB", Ratio: 1.0},
		{PrefName: "PrefC", Ratio: 2.0},
		{PrefName: "PrefD", Ratio: 3.0},
		{PrefName: "PrefE", Ratio: 4.0},
		{PrefName: "PrefF", Ratio: 5.0},
		{PrefName: "PrefG", Ratio: 6.0},
		{PrefName: "PrefH", Ratio: 7.0},
		{PrefName: "PrefI", Ratio: 8.0},
		{PrefName: "PrefJ", Ratio: 9.0},
		{PrefName: "PrefK", Ratio: 10.0},
		{PrefName: "PrefL", Ratio: 11.0},
		{PrefName: "PrefM", Ratio: 12.0},
		// {PrefName: "PrefN", Ratio: 13.0},
		// {PrefName: "PrefO", Ratio: 14.0},
		// {PrefName: "PrefP", Ratio: 15.0},
	}

	_ = rankingRecordList

	for _, record := range rankingRecordList {
		slog.Info("Before AssignTiers", "PrefName", record.PrefName, "Ratio", record.Ratio, "Tier", record.Tier)
	}

	rankingRecordList.AssignTiers()

	for _, record := range rankingRecordList {
		slog.Info("After AssignTiers", "PrefName", record.PrefName, "Ratio", record.Ratio, "Tier", record.Tier)
	}
}
