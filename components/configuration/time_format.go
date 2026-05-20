package configuration

import (
	"time"
)

func FormatTime(utcStr string) string {
	t, err := time.Parse(time.RFC3339, utcStr)
	if err != nil {
		return utcStr
	}
	loc, err := time.LoadLocation(Timezone())
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}
