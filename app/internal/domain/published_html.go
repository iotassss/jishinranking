package domain

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

type PublishedHTML string

func (p PublishedHTML) UpdatePeriod(from time.Time, to time.Time) PublishedHTML {
	// 集計期間: YYYY-MM-DD ～ YYYY-MM-DD 形式の部分を更新
	rePeriod := regexp.MustCompile(`集計期間: \d{4}-\d{2}-\d{2} ～ \d{4}-\d{2}-\d{2}`)
	updated := rePeriod.ReplaceAllString(string(p), fmt.Sprintf(`集計期間: %s ～ %s`, from.Format("2006-01-02"), to.Format("2006-01-02")))

	slog.Info("Updated Period", "changed", updated != string(p))

	return PublishedHTML(updated)
}

// この関数が適切にHTMLを更新できていない
func (p PublishedHTML) UpdateUpdatedAt(t time.Time) PublishedHTML {
	src := string(p)

	// id="updated-at" の time 要素ブロック全体を取り出す（属性順・改行に強い）
	blockRe := regexp.MustCompile(`(?s)<time\b[^>]*\bid="updated-at"[^>]*>.*?</time>`)
	block := blockRe.FindString(src)
	if block == "" {
		slog.Warn("updated-at time tag not found")
		return p
	}

	ts := t.Format(time.RFC3339)
	// 既存HTMLが &#43; を使っているので揃える（不要ならこの1行を消してOK）
	tsAttr := strings.ReplaceAll(ts, "+", "&#43;")

	// datetime / data-now を更新（data-now が無くてもOK）
	datetimeRe := regexp.MustCompile(`\bdatetime="[^"]*"`)
	datanowRe := regexp.MustCompile(`\bdata-now="[^"]*"`)

	updatedBlock := datetimeRe.ReplaceAllString(block, fmt.Sprintf(`datetime="%s"`, tsAttr))
	updatedBlock = datanowRe.ReplaceAllString(updatedBlock, fmt.Sprintf(`data-now="%s"`, tsAttr))

	// inner text も更新（>...</time> の ... 部分）
	innerRe := regexp.MustCompile(`>([^<]*)</time>`)
	updatedBlock = innerRe.ReplaceAllString(updatedBlock, fmt.Sprintf(`>%s</time>`, tsAttr))

	out := blockRe.ReplaceAllString(src, updatedBlock)

	slog.Info("Updated UpdatedAt", "updated_at", ts, "changed", out != src)
	return PublishedHTML(out)
}
