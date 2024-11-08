package crm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type DateTimeYMDHMS time.Time
type TimeHMS time.Time
type DateYMD time.Time

func (j *DateTimeYMDHMS) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*j = DateTimeYMDHMS(t)
	return nil
}

func (j *DateTimeYMDHMS) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*j).Format("2006-01-02 15:04:05"))
}

func (d *DateYMD) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*d = DateYMD(t)
	return nil
}

func (d *DateYMD) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*d).Format("2006-01-02"))
}

func (d *DateYMD) Sub(other DateYMD) time.Duration {
	return time.Time(*d).Sub(time.Time(other))
}

func (d *DateYMD) SubTime(other time.Time) time.Duration {
	return time.Time(*d).Sub(other)
}

func (t *TimeHMS) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	formats := []string{"15:04:05", "15:04"}
	var parsedTime time.Time
	var err error
	for _, format := range formats {
		parsedTime, err = time.Parse(format, s)
		if err == nil {
			*t = TimeHMS(parsedTime)
			return nil
		}
	}

	return fmt.Errorf("не удалось распарсить время: %s", s)
}

func (t *TimeHMS) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*t).Format("15:04:05"))
}

func parseJSONFile(target interface{}, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		panic("Error opening file")
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err := json.NewDecoder(file).Decode(&target); err != nil {
		panic(fmt.Errorf("error decoding JSON: %w", err))
	}
}

func mergeToDatetime(date time.Time, time_ time.Time) time.Time {
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		time_.Hour(), time_.Minute(), time_.Second(), 0, time_.Location(),
	)
}
