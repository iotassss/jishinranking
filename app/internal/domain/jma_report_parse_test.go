package domain

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func TestParseJMAReports(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     struct {
			eventID          string
			editorialOffice  string
			hypocenterName   string
			hypocenterCode   string
			coordinate       string
			magnitude        string
			magnitudeDesc    string
			maxIntensity     string
			prefCount        int
			firstPrefName    string
			firstPrefCode    string
			firstAreaName    string
			firstCityName    string
			firstStationName string
			tsunamiComment   string
		}
	}{
		{
			name:     "Simple case - single prefecture, single station",
			filename: "20260117234735_0_VXSE53_270000.xml",
			want: struct {
				eventID          string
				editorialOffice  string
				hypocenterName   string
				hypocenterCode   string
				coordinate       string
				magnitude        string
				magnitudeDesc    string
				maxIntensity     string
				prefCount        int
				firstPrefName    string
				firstPrefCode    string
				firstAreaName    string
				firstCityName    string
				firstStationName string
				tsunamiComment   string
			}{
				eventID:          "20260118084432",
				editorialOffice:  "大阪管区気象台",
				hypocenterName:   "トカラ列島近海",
				hypocenterCode:   "798",
				coordinate:       "+29.4+129.6-10000/",
				magnitude:        "2.1",
				magnitudeDesc:    "Ｍ２．１",
				maxIntensity:     "1",
				prefCount:        1,
				firstPrefName:    "鹿児島県",
				firstPrefCode:    "46",
				firstAreaName:    "鹿児島県十島村",
				firstCityName:    "鹿児島十島村",
				firstStationName: "鹿児島十島村悪石島＊",
				tsunamiComment:   "この地震による津波の心配はありません。",
			},
		},
		{
			name:     "Complex case - multiple prefectures, multiple stations",
			filename: "20260117184909_0_VXSE53_010000.xml",
			want: struct {
				eventID          string
				editorialOffice  string
				hypocenterName   string
				hypocenterCode   string
				coordinate       string
				magnitude        string
				magnitudeDesc    string
				maxIntensity     string
				prefCount        int
				firstPrefName    string
				firstPrefCode    string
				firstAreaName    string
				firstCityName    string
				firstStationName string
				tsunamiComment   string
			}{
				eventID:          "20260118034643",
				editorialOffice:  "気象庁本庁",
				hypocenterName:   "岐阜県飛騨地方",
				hypocenterCode:   "430",
				coordinate:       "+36.0+137.4-10000/",
				magnitude:        "3.7",
				magnitudeDesc:    "Ｍ３．７",
				maxIntensity:     "2",
				prefCount:        2,
				firstPrefName:    "岐阜県",
				firstPrefCode:    "21",
				firstAreaName:    "岐阜県飛騨",
				firstCityName:    "高山市",
				firstStationName: "高山市丹生川町坊方＊",
				tsunamiComment:   "この地震による津波の心配はありません。",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", path, err)
			}

			var report Report
			err = xml.Unmarshal(data, &report)
			if err != nil {
				t.Fatalf("Failed to unmarshal XML: %v", err)
			}

			// Head の検証
			if report.Head.EventID != tt.want.eventID {
				t.Errorf("Head.EventID = %q, want %q", report.Head.EventID, tt.want.eventID)
			}

			// Control の検証
			if report.Control.EditorialOffice != tt.want.editorialOffice {
				t.Errorf("Control.EditorialOffice = %q, want %q", report.Control.EditorialOffice, tt.want.editorialOffice)
			}

			// Earthquake の検証
			if report.Body.Earthquake == nil {
				t.Fatal("Body.Earthquake is nil")
			}

			// Hypocenter の検証
			if report.Body.Earthquake.Hypocenter == nil {
				t.Fatal("Hypocenter is nil")
			}
			if report.Body.Earthquake.Hypocenter.Area == nil {
				t.Fatal("Hypocenter.Area is nil")
			}
			if report.Body.Earthquake.Hypocenter.Area.Name != tt.want.hypocenterName {
				t.Errorf("Hypocenter.Area.Name = %q, want %q", report.Body.Earthquake.Hypocenter.Area.Name, tt.want.hypocenterName)
			}
			if report.Body.Earthquake.Hypocenter.Area.Code != tt.want.hypocenterCode {
				t.Errorf("Hypocenter.Area.Code = %q, want %q", report.Body.Earthquake.Hypocenter.Area.Code, tt.want.hypocenterCode)
			}
			if report.Body.Earthquake.Hypocenter.Area.Coordinate != tt.want.coordinate {
				t.Errorf("Hypocenter.Area.Coordinate = %q, want %q", report.Body.Earthquake.Hypocenter.Area.Coordinate, tt.want.coordinate)
			}

			// Magnitude の検証
			if report.Body.Earthquake.Magnitude == nil {
				t.Fatal("Magnitude is nil")
			}
			if report.Body.Earthquake.Magnitude.Value != tt.want.magnitude {
				t.Errorf("Magnitude.Value = %q, want %q", report.Body.Earthquake.Magnitude.Value, tt.want.magnitude)
			}
			if report.Body.Earthquake.Magnitude.Description != tt.want.magnitudeDesc {
				t.Errorf("Magnitude.Description = %q, want %q", report.Body.Earthquake.Magnitude.Description, tt.want.magnitudeDesc)
			}

			// Intensity の検証
			if report.Body.Intensity == nil {
				t.Fatal("Intensity is nil")
			}
			if report.Body.Intensity.Observation == nil {
				t.Fatal("Intensity.Observation is nil")
			}
			if report.Body.Intensity.Observation.MaxInt != tt.want.maxIntensity {
				t.Errorf("Intensity.Observation.MaxInt = %q, want %q", report.Body.Intensity.Observation.MaxInt, tt.want.maxIntensity)
			}

			// Prefecture の検証
			if len(report.Body.Intensity.Observation.Prefs) != tt.want.prefCount {
				t.Errorf("Prefecture count = %d, want %d", len(report.Body.Intensity.Observation.Prefs), tt.want.prefCount)
			}
			if len(report.Body.Intensity.Observation.Prefs) > 0 {
				pref := report.Body.Intensity.Observation.Prefs[0]
				if pref.Name != tt.want.firstPrefName {
					t.Errorf("Pref[0].Name = %q, want %q", pref.Name, tt.want.firstPrefName)
				}
				if pref.Code != tt.want.firstPrefCode {
					t.Errorf("Pref[0].Code = %q, want %q", pref.Code, tt.want.firstPrefCode)
				}

				// Area の検証
				if len(pref.Areas) > 0 {
					area := pref.Areas[0]
					if area.Name != tt.want.firstAreaName {
						t.Errorf("Area[0].Name = %q, want %q", area.Name, tt.want.firstAreaName)
					}

					// City の検証
					if len(area.Cities) > 0 {
						city := area.Cities[0]
						if city.Name != tt.want.firstCityName {
							t.Errorf("City[0].Name = %q, want %q", city.Name, tt.want.firstCityName)
						}

						// Station の検証
						if len(city.IntensityStations) > 0 {
							station := city.IntensityStations[0]
							if station.Name != tt.want.firstStationName {
								t.Errorf("Station[0].Name = %q, want %q", station.Name, tt.want.firstStationName)
							}
						}
					}
				}
			}

			// Comments の検証
			if report.Body.Comments == nil {
				t.Fatal("Comments is nil")
			}
			if report.Body.Comments.ForecastComment == nil {
				t.Fatal("ForecastComment is nil")
			}
			if report.Body.Comments.ForecastComment.Text != tt.want.tsunamiComment {
				t.Errorf("ForecastComment.Text = %q, want %q", report.Body.Comments.ForecastComment.Text, tt.want.tsunamiComment)
			}
		})
	}
}
