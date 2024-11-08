package bot

import (
	"time"
)

func GetMapValues(m map[string]string) []string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func DatetimeToDate(datetime time.Time) time.Time {
	return time.Date(
		datetime.Year(), datetime.Month(), datetime.Day(),
		0, 0, 0, 0, datetime.Location(),
	)
}

func DatetimeToTime(datetime time.Time) time.Time {
	return time.Date(0, 1, 1,
		datetime.Hour(), datetime.Minute(), 0, 0, datetime.Location())
}
