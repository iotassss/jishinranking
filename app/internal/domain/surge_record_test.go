package domain

import (
	"math"
	"testing"
	"time"
)

var testNow = time.Date(2026, 4, 6, 9, 31, 47, 0, time.UTC)

// eq は EarthquakeRecord を簡潔に組み立てるヘルパー。
func eq(occurredAt time.Time, prefs []PrefIntensityDetail) EarthquakeRecord {
	return EarthquakeRecord{
		OccurredAt:    occurredAt,
		ObservedPrefs: prefs,
	}
}

// pref は PrefIntensityDetail を簡潔に組み立てるヘルパー。
func pref(code, name, maxInt string) PrefIntensityDetail {
	return PrefIntensityDetail{Code: code, Name: name, MaxInt: maxInt}
}

// daysAgo は testNow から d 日前の時刻を返す。
func daysAgo(d float64) time.Time {
	return testNow.Add(-time.Duration(d * float64(24*time.Hour)))
}

// ---- intensityWeight -------------------------------------------------------

func TestIntensityWeight(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1", 1},
		{"2", 2},
		{"3", 4},
		{"4", 8},
		{"5-", 16},
		{"5+", 24},
		{"6-", 40},
		{"6+", 64},
		{"7", 100},
		{"", 0},
		{"不明", 0},
		{"-", 0},
	}
	for _, c := range cases {
		got := intensityWeight(c.in)
		if got != c.want {
			t.Errorf("intensityWeight(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// ---- calcDecayScore --------------------------------------------------------

func TestCalcDecayScore_Empty(t *testing.T) {
	score := calcDecayScore(nil, testNow, 14.0)
	if score != 0 {
		t.Errorf("empty events: got %v, want 0", score)
	}
}

func TestCalcDecayScore_FutureEventsIgnored(t *testing.T) {
	events := []surgeEvent{
		{OccurredAt: testNow.Add(1 * time.Hour), Weight: 100},
	}
	score := calcDecayScore(events, testNow, 14.0)
	if score != 0 {
		t.Errorf("future event: got %v, want 0", score)
	}
}

func TestCalcDecayScore_ExactlyNow(t *testing.T) {
	events := []surgeEvent{
		{OccurredAt: testNow, Weight: 8},
	}
	score := calcDecayScore(events, testNow, 14.0)
	// days=0 → exp(0)=1 → score=8
	if math.Abs(score-8.0) > 1e-9 {
		t.Errorf("event at now: got %v, want 8", score)
	}
}

func TestCalcDecayScore_OneTauDecay(t *testing.T) {
	tau := 7.0
	events := []surgeEvent{
		{OccurredAt: daysAgo(tau), Weight: 10},
	}
	score := calcDecayScore(events, testNow, tau)
	want := 10.0 * math.Exp(-1) // e^-1
	if math.Abs(score-want) > 1e-9 {
		t.Errorf("1τ decay: got %v, want %v", score, want)
	}
}

func TestCalcDecayScore_MultipleEvents(t *testing.T) {
	events := []surgeEvent{
		{OccurredAt: daysAgo(0), Weight: 4},
		{OccurredAt: daysAgo(3.5), Weight: 8},
	}
	tau := 7.0
	want := 4.0*math.Exp(0) + 8.0*math.Exp(-0.5)
	got := calcDecayScore(events, testNow, tau)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("multi events: got %v, want %v", got, want)
	}
}

// ---- MakeSurgeRecordList ---------------------------------------------------

func TestMakeSurgeRecordList_Empty(t *testing.T) {
	result := MakeSurgeRecordList(EarthquakeRecordList{}, testNow, 20.0, 2.0, 10)
	if len(result) != 0 {
		t.Errorf("empty input: got %d records, want 0", len(result))
	}
}

func TestMakeSurgeRecordList_ZeroLimit(t *testing.T) {
	eqs := EarthquakeRecordList{
		eq(daysAgo(0), []PrefIntensityDetail{pref("01", "北海道", "4")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 0, 0, 0)
	if len(result) != 0 {
		t.Errorf("zero limit: got %d records, want 0", len(result))
	}
}

// 急上昇しない都道府県（長期実績あり・短期スコア低い）はランキング外になる。
func TestMakeSurgeRecordList_NoSurge(t *testing.T) {
	// 過去だけに均等分布 → short ≒ normalizedLong → ratio < 2
	eqs := EarthquakeRecordList{
		eq(daysAgo(3), []PrefIntensityDetail{pref("01", "北海道", "3")}),
		eq(daysAgo(5), []PrefIntensityDetail{pref("01", "北海道", "3")}),
		eq(daysAgo(7), []PrefIntensityDetail{pref("01", "北海道", "3")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 0.5, 2.0, 10)
	if len(result) != 0 {
		t.Errorf("no surge: got %d records, want 0", len(result))
	}
}

// 短期スコアが minShortScore 未満の場合は除外される。
func TestMakeSurgeRecordList_BelowMinShortScore(t *testing.T) {
	// 震度1 (weight=1) が 1件だけ → short ≈ 1 < 5
	eqs := EarthquakeRecordList{
		eq(daysAgo(0.1), []PrefIntensityDetail{pref("01", "北海道", "1")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)
	if len(result) != 0 {
		t.Errorf("below min short score: got %d records, want 0", len(result))
	}
}

// 急上昇する都道府県がランキングに含まれる。
func TestMakeSurgeRecordList_SurgeDetected(t *testing.T) {
	// 直近にだけ震度4の地震が集中（過去実績なし）→ ratio >> 2, short > 5
	eqs := EarthquakeRecordList{
		eq(daysAgo(0.1), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.3), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.5), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.8), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(1.1), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)
	if len(result) != 1 {
		t.Fatalf("surge detected: got %d records, want 1", len(result))
	}
	if result[0].PrefCode != "30" {
		t.Errorf("pref code: got %q, want %q", result[0].PrefCode, "30")
	}
	if result[0].Rank != 1 {
		t.Errorf("rank: got %d, want 1", result[0].Rank)
	}
	if result[0].Score <= 0 {
		t.Errorf("score: got %v, want > 0", result[0].Score)
	}
}

// 複数都道府県がそれぞれ独立に評価される。
func TestMakeSurgeRecordList_MultiplePrefectures(t *testing.T) {
	// 和歌山は直近にM大、石川は少し後れを取る
	eqs := EarthquakeRecordList{
		// 和歌山: 直近5件 震度4
		eq(daysAgo(0.1), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.2), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.3), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.5), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.8), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		// 石川: 直近3件 震度4
		eq(daysAgo(0.2), []PrefIntensityDetail{pref("17", "石川県", "4")}),
		eq(daysAgo(0.5), []PrefIntensityDetail{pref("17", "石川県", "4")}),
		eq(daysAgo(1.0), []PrefIntensityDetail{pref("17", "石川県", "4")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)

	if len(result) != 2 {
		t.Fatalf("got %d records, want 2", len(result))
	}
	// 和歌山がトップ
	if result[0].PrefCode != "30" {
		t.Errorf("rank1 pref: got %q, want 30", result[0].PrefCode)
	}
	if result[1].PrefCode != "17" {
		t.Errorf("rank2 pref: got %q, want 17", result[1].PrefCode)
	}
	if result[0].Score <= result[1].Score {
		t.Errorf("score order: rank1=%v should be > rank2=%v", result[0].Score, result[1].Score)
	}
}

// limit が結果件数を制限する。
func TestMakeSurgeRecordList_LimitApplied(t *testing.T) {
	// 3都道府県すべてで急上昇を起こす
	eqs := make(EarthquakeRecordList, 0)
	prefs := []struct{ code, name string }{
		{"01", "北海道"},
		{"13", "東京都"},
		{"27", "大阪府"},
	}
	for _, p := range prefs {
		for i := 0; i < 5; i++ {
			eqs = append(eqs, eq(daysAgo(float64(i)*0.2+0.1), []PrefIntensityDetail{pref(p.code, p.name, "4")}))
		}
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 2)
	if len(result) != 2 {
		t.Errorf("limit=2: got %d records, want 2", len(result))
	}
}

// ランクが連番で正しく割り当てられる。
func TestMakeSurgeRecordList_RanksAssigned(t *testing.T) {
	eqs := make(EarthquakeRecordList, 0)
	for _, code := range []string{"30", "17", "09"} {
		for i := 0; i < 5; i++ {
			eqs = append(eqs, eq(daysAgo(float64(i)*0.2), []PrefIntensityDetail{pref(code, code, "4")}))
		}
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)
	for i, r := range result {
		if r.Rank != i+1 {
			t.Errorf("index %d: Rank=%d, want %d", i, r.Rank, i+1)
		}
	}
}

// 未知の震度（maxInt=""）はスキップされる。
func TestMakeSurgeRecordList_UnknownIntensityIgnored(t *testing.T) {
	eqs := EarthquakeRecordList{
		eq(daysAgo(0.1), []PrefIntensityDetail{pref("01", "北海道", "")}),
		eq(daysAgo(0.2), []PrefIntensityDetail{pref("01", "北海道", "-")}),
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)
	if len(result) != 0 {
		t.Errorf("unknown intensity: got %d records, want 0", len(result))
	}
}

// TodayCount は過去24h以内の地震件数を正確に反映する。
func TestMakeSurgeRecordList_TodayCount(t *testing.T) {
	// 5件中 3件が24h以内、2件が24h以降
	eqs := EarthquakeRecordList{
		eq(daysAgo(0.1), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.3), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(0.9), []PrefIntensityDetail{pref("30", "和歌山県", "4")}),
		eq(daysAgo(1.5), []PrefIntensityDetail{pref("30", "和歌山県", "4")}), // 24h超
		eq(daysAgo(2.0), []PrefIntensityDetail{pref("30", "和歌山県", "4")}), // 24h超
	}
	result := MakeSurgeRecordList(eqs, testNow, 5.0, 2.0, 10)
	if len(result) == 0 {
		t.Fatal("expected at least 1 surge record")
	}
	if result[0].TodayCount != 3 {
		t.Errorf("TodayCount: got %d, want 3", result[0].TodayCount)
	}
}

// SortByScoreDesc はスコア降順に並び替え、同スコア時は TodayCount 降順・PrefCode 昇順になる。
func TestSortByScoreDesc(t *testing.T) {
	list := SurgeRecordList{
		{PrefCode: "30", Score: 50.0, TodayCount: 3},
		{PrefCode: "17", Score: 80.0, TodayCount: 5},
		{PrefCode: "09", Score: 80.0, TodayCount: 7},
		{PrefCode: "01", Score: 20.0, TodayCount: 1},
	}
	list.SortByScoreDesc()
	// score=80 が先、同スコアは TodayCount 降順なので "09"(7)→"17"(5)
	if list[0].PrefCode != "09" {
		t.Errorf("index0: got %q, want 09", list[0].PrefCode)
	}
	if list[1].PrefCode != "17" {
		t.Errorf("index1: got %q, want 17", list[1].PrefCode)
	}
	if list[2].PrefCode != "30" {
		t.Errorf("index2: got %q, want 30", list[2].PrefCode)
	}
	if list[3].PrefCode != "01" {
		t.Errorf("index3: got %q, want 01", list[3].PrefCode)
	}
}

// AssignRanks は 1 始まりの連番を付与する。
func TestAssignRanks(t *testing.T) {
	list := SurgeRecordList{
		{PrefCode: "09"},
		{PrefCode: "17"},
		{PrefCode: "30"},
	}
	list.AssignRanks()
	for i, r := range list {
		if r.Rank != i+1 {
			t.Errorf("index %d: Rank=%d, want %d", i, r.Rank, i+1)
		}
	}
}
