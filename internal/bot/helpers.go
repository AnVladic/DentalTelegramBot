package bot

import (
	"sort"
	"strings"
	"time"
)

func GetMapValues(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(m))
	for _, k := range keys {
		values = append(values, m[k])
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

func IsMatchIgnoreCase(text string, phrases []string) bool {
	text = strings.ToLower(text)
	for _, phrase := range phrases {
		if text == strings.ToLower(phrase) {
			return true
		}
	}
	return false
}
