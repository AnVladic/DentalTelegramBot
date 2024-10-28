package crm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type IDentalProClient interface {
	Timesheet(startDate, endDate time.Time) ([]TimesheetResponse, error)
}

type DentalProClient struct {
	Token     string
	SecretKey string
}

type TimesheetResponse struct {
	Id           int64     `json:"id"`
	PlannedStart time.Time `json:"plannedStart"`
	PlannedEnd   time.Time `json:"plannedEnd"`
	ActualStart  time.Time `json:"actualStart"`
	ActualEnd    time.Time `json:"actualEnd"`
	UserID       int64     `json:"userID"`
	BranchID     int64     `json:"branchID"`
}

func NewDentalProClient(token string, secretKey string, test bool) IDentalProClient {
	if test {
		return &DentalProClientTest{Token: token, SecretKey: secretKey}
	}
	return &DentalProClient{Token: token, SecretKey: secretKey}
}

func (c *DentalProClient) baseURL() string {
	return "https://olimp.crm3.dental-pro.online"
}

func (c *DentalProClient) ConvertDateToStr(date time.Time) string {
	return date.Format("2006-01-02")
}

func (c *DentalProClient) Timesheet(startDate, endDate time.Time) ([]TimesheetResponse, error) {
	baseURL := c.baseURL() + "/api/timesheet/list"
	query := url.Values{}
	query.Add("token", c.Token)
	query.Add("secret", c.SecretKey)

	jsonData, err := json.Marshal(map[string]string{
		"date_start": c.ConvertDateToStr(startDate),
		"date_end":   c.ConvertDateToStr(endDate),
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		fmt.Sprintf("%s?%s", baseURL, query.Encode()),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("error sending POST request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: received status code %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var responseData []TimesheetResponse
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return responseData, nil
}
