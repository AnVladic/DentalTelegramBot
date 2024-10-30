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
	DoctorsList() ([]Doctor, error)
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

type Doctor struct {
	ID           int64             `json:"id"`
	UserID       int64             `json:"user_id"`
	IsVIP        bool              `json:"is_vip"`
	DateAdded    time.Time         `json:"date_added"`
	DateDelete   *time.Time        `json:"date_delete"`
	IsHidden     bool              `json:"is_hidden"`
	MoneyPerHour int               `json:"money_per_hour"`
	Branches     map[string]string `json:"branches"`
	Name         string            `json:"name"`
	Surname      string            `json:"surname"`
	SecondName   string            `json:"second_name"`
	FIO          string            `json:"fio"`
	UserGroups   map[string]string `json:"user_groups"`
	Departments  map[string]string `json:"departments"`
	Photo        *string           `json:"photo"`
	Phone        string            `json:"phone"`
}

func NewDentalProClient(token string, secretKey string, test bool) IDentalProClient {
	if test {
		return NewDentalProClientTest(token, secretKey)
	}
	return &DentalProClient{Token: token, SecretKey: secretKey}
}

func (c *DentalProClient) baseURL() string {
	return "https://olimp.crm3.dental-pro.online"
}

func (c *DentalProClient) ConvertDateToStr(date time.Time) string {
	return date.Format("2006-01-02")
}

func (c *DentalProClient) DoctorsList() ([]Doctor, error) {
	return nil, nil
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

func GetDoctorByID(doctors []Doctor, doctorID int64) *Doctor {
	for _, doctor := range doctors {
		if doctor.ID == doctorID {
			return &doctor
		}
	}
	return nil
}
