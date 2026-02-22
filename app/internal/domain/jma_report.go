package domain

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

const (
	// 震度速報
	TelegramCodeEarthquakeBreaking = "VXSE51"
	// 地震情報（震源に関する情報）
	TelegramCodeEarthquakeEpicenter = "VXSE52"
	// 地震情報（震源・震度に関する情報）
	TelegramCodeEarthquakeDetail = "VXSE53"
)

// JMATime parses RFC3339 like "2025-12-20T06:48:56+09:00"
type JMATime struct{ time.Time }

func (t *JMATime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("parse time %q: %w", s, err)
	}
	t.Time = tt
	return nil
}

type Report struct {
	XMLName  xml.Name `xml:"Report" json:"-"`
	Control  Control  `xml:"Control" json:"control"`
	Head     Head     `xml:"Head" json:"head"`
	Body     Body     `xml:"Body" json:"body"`
	Metadata Metadata `json:"metadata"`
}

// jmaのxmlに含まれない追加情報
type Metadata struct {
	URL string `json:"url"`
}

type Control struct {
	Title            string  `xml:"Title" json:"title"`
	DateTime         JMATime `xml:"DateTime" json:"dateTime"`
	Status           string  `xml:"Status" json:"status"`
	EditorialOffice  string  `xml:"EditorialOffice" json:"editorialOffice"`
	PublishingOffice string  `xml:"PublishingOffice" json:"publishingOffice"`
}

type Head struct {
	Title           string   `xml:"Title" json:"title"`
	ReportDateTime  JMATime  `xml:"ReportDateTime" json:"reportDateTime"`
	TargetDateTime  JMATime  `xml:"TargetDateTime" json:"targetDateTime"`
	EventID         string   `xml:"EventID" json:"eventID"`
	InfoType        string   `xml:"InfoType" json:"infoType"`
	Serial          string   `xml:"Serial" json:"serial"`
	InfoKind        string   `xml:"InfoKind" json:"infoKind"`
	InfoKindVersion string   `xml:"InfoKindVersion" json:"infoKindVersion"`
	Headline        Headline `xml:"Headline" json:"headline"`
}

type Headline struct {
	Text         string        `xml:"Text" json:"text"`
	Informations []Information `xml:"Information" json:"informations"`
}

type Information struct {
	Type  string `xml:"type,attr" json:"type"`
	Items []Item `xml:"Item" json:"items"`
}

type Item struct {
	Kind  Kind  `xml:"Kind" json:"kind"`
	Areas Areas `xml:"Areas" json:"areas"`
}

type Kind struct {
	Name string `xml:"Name" json:"name"`
}

type Areas struct {
	CodeType string `xml:"codeType,attr" json:"codeType"`
	AreaList []Area `xml:"Area" json:"areaList"`
}

// Areaは既存のIntensity用Areaと同名なので、必要に応じてリネームしてください
// ここでは同じ構造なので流用します

type Body struct {
	Earthquake *Earthquake `xml:"Earthquake" json:"earthquake"`
	Intensity  *struct {
		Observation *struct {
			CodeDefine *struct {
				Types []struct {
					XPath string `xml:"xpath,attr" json:"xpath"`
					Name  string `xml:",chardata" json:"name"`
				} `xml:"Type" json:"types"`
			} `xml:"CodeDefine" json:"codeDefine"`
			MaxInt string `xml:"MaxInt" json:"maxInt"`
			Prefs  []struct {
				Name   string `xml:"Name" json:"name"`
				Code   string `xml:"Code" json:"code"`
				MaxInt string `xml:"MaxInt" json:"maxInt"`
				Areas  []struct {
					Name   string `xml:"Name" json:"name"`
					Code   string `xml:"Code" json:"code"`
					MaxInt string `xml:"MaxInt" json:"maxInt"`
					Cities []struct {
						Name              string `xml:"Name" json:"name"`
						Code              string `xml:"Code" json:"code"`
						MaxInt            string `xml:"MaxInt" json:"maxInt"`
						IntensityStations []struct {
							Name string `xml:"Name" json:"name"`
							Code string `xml:"Code" json:"code"`
							Int  string `xml:"Int" json:"int"`
						} `xml:"IntensityStation" json:"intensityStations"`
					} `xml:"City" json:"cities"`
				} `xml:"Area" json:"areas"`
			} `xml:"Pref" json:"prefs"`
		} `xml:"Observation" json:"observation"`
	} `xml:"Intensity" json:"intensity"`
	Comments *struct {
		ForecastComment *struct {
			CodeType string `xml:"codeType,attr" json:"codeType"`
			Text     string `xml:"Text" json:"text"`
			Code     string `xml:"Code" json:"code"`
		} `xml:"ForecastComment" json:"forecastComment"`
		VarComment *struct {
			CodeType string `xml:"codeType,attr" json:"codeType"`
			Text     string `xml:"Text" json:"text"`
			Code     string `xml:"Code" json:"code"`
		} `xml:"VarComment" json:"varComment"`
	} `xml:"Comments" json:"comments"`
}

type Earthquake struct {
	OriginTime  JMATime `xml:"OriginTime" json:"originTime"`   // ★発生時刻
	ArrivalTime JMATime `xml:"ArrivalTime" json:"arrivalTime"` // (あれば)
	Condition   string  `xml:"Condition" json:"condition"`

	Hypocenter *struct {
		Area *struct {
			Name       string `xml:"Name" json:"name"`
			Code       string `xml:"Code" json:"code"`
			Coordinate string `xml:"Coordinate" json:"coordinate"`
		} `xml:"Area" json:"area"`
		Location *struct {
			Latitude  string `xml:"Latitude" json:"latitude"`
			Longitude string `xml:"Longitude" json:"longitude"`
			Depth     string `xml:"Depth" json:"depth"`
		} `xml:"Location" json:"location"`
	} `xml:"Hypocenter" json:"hypocenter"`

	Magnitude *struct {
		Type        string `xml:"type,attr" json:"type"`
		Description string `xml:"description,attr" json:"description"`
		Value       string `xml:",chardata" json:"value"`
	} `xml:"Magnitude" json:"magnitude"`
}

type CodeDefine struct {
	Types []TypeDef `xml:"Type" json:"types"`
}

type TypeDef struct {
	XPath string `xml:"xpath,attr" json:"xpath"`
	Name  string `xml:",chardata" json:"name"`
}

type Pref struct {
	Name   string `xml:"Name" json:"name"`
	Code   string `xml:"Code" json:"code"`
	MaxInt string `xml:"MaxInt" json:"maxInt"`
	Areas  []Area `xml:"Area" json:"areas"`
}

type Area struct {
	Name   string `xml:"Name" json:"name"`
	Code   string `xml:"Code" json:"code"`
	MaxInt string `xml:"MaxInt" json:"maxInt"`
}

type ReportKey struct {
	EventID string // 無い場合は空文字
	Title   string
	Office  string
	Status  string
}

type ReportList []Report

func (rl ReportList) Latest() Report {
	var latest Report
	var latestTime time.Time
	for _, report := range rl {
		t := report.Head.ReportDateTime.Time
		if t.After(latestTime) {
			latest = report
			latestTime = t
		}
	}
	return latest
}

func (rl ReportList) ToMap() map[ReportKey]Report {
	m := make(map[ReportKey]Report)
	for _, r := range rl {
		key := ReportKey{
			EventID: r.Head.EventID,
			Title:   r.Control.Title,
			Office:  r.Control.PublishingOffice,
			Status:  r.Control.Status,
		}
		m[key] = r
	}
	return m
}

// Reportに記載されるに日時は、ReportedDatetime, TargetDatetime, OriginTimeなど複数ある
// ここではReportedDatetimeを基準にしてフィルタリングする
// 理由: TargetDatetime, OriginTimeは非必須の可能性があるため。本来はOriginTimeを基準にしたい
func (rl ReportList) NewerThan(targetReport Report) ReportList {
	var filtered ReportList
	for _, r := range rl {
		if r.Head.ReportDateTime.Time.After(targetReport.Head.ReportDateTime.Time) {
			filtered = append(filtered, r)
		}

		if r.Head.ReportDateTime.Time.Equal(targetReport.Head.ReportDateTime.Time) {
			// 同時刻でもEventIDが異なる場合は戻り値に含める
			if r.Head.EventID != targetReport.Head.EventID {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// 重複を排除してマージする
// 順番は保証しない
func (rl ReportList) Merge(other ReportList) ReportList {
	merged := rl.ToMap()
	for _, r := range other {
		key := ReportKey{
			EventID: r.Head.EventID,
			Title:   r.Control.Title,
			Office:  r.Control.PublishingOffice,
			Status:  r.Control.Status,
		}
		if existing, exists := merged[key]; exists {
			// より新しいReportDateTimeのものを採用する
			if r.Head.ReportDateTime.Time.After(existing.Head.ReportDateTime.Time) {
				merged[key] = r
			}
		} else {
			merged[key] = r
		}
	}

	var result ReportList
	for _, r := range merged {
		result = append(result, r)
	}

	return result
}

func (rl ReportList) Between(from, to time.Time) ReportList {
	var filtered ReportList
	for _, r := range rl {
		if r.Head.ReportDateTime.Time.After(from) && r.Head.ReportDateTime.Time.Before(to) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (rl ReportList) FilterByTelegramCode(code string) ReportList {
	if strings.TrimSpace(code) == "" {
		return rl
	}

	var filtered ReportList
	for _, r := range rl {
		if strings.Contains(strings.ToUpper(r.Metadata.URL), strings.ToUpper(code)) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
