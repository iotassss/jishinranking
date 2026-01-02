package domain

import (
	"time"
)

type ReportPeriod struct {
	Start time.Time
	End   time.Time
}

func (rp ReportPeriod) String() string {
	return rp.Start.Format("2006-01-02") + " ～ " + rp.End.Format("2006-01-02")
}
