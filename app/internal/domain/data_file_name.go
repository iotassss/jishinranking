package domain

import "time"

const (
	DataFileNameFormat = "20060102T150405Z.json"
)

type DataFileName string

func NewDataFileNameFromTime(time time.Time) DataFileName {
	return DataFileName(time.UTC().Format(DataFileNameFormat))
}

func (d DataFileName) String() string {
	return string(d)
}

func (d DataFileName) ToTime() (time.Time, error) {
	return time.Parse(DataFileNameFormat, string(d))
}
