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
	DoctorWorkSchedule(date time.Time, doctorID int64) ([]WorkSchedule, error)
	AvailableAppointments(
		userID int64, doctorIDS []int64, isPlanned bool) (map[int64]map[int64]Appointment, error)

	CreatePatient(name, surname string, phone string) (Patient, error)
	PatientByPhone(phone string) (Patient, error)
}

type DentalProClient struct {
	Token     string
	SecretKey string
}

type TimesheetResponse struct {
	ID           int64     `json:"id"`
	PlannedStart time.Time `json:"plannedStart"`
	PlannedEnd   time.Time `json:"plannedEnd"`
	ActualStart  time.Time `json:"actualStart"`
	ActualEnd    time.Time `json:"actualEnd"`
	UserID       int64     `json:"userID"`
	BranchID     int64     `json:"branchID"`
}

type Appointment struct {
	ID             int     `json:"id"`
	Cost           float64 `json:"cost"`
	Name           string  `json:"name"`
	Time           int     `json:"time"`
	Color          string  `json:"color"`
	DiagnosticType string  `json:"diagnosticType"`
	IsPlanned      bool    `json:"isPlanned"`
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

type WorkSchedule struct {
	TimeStart *time.Time `json:"time_start"`
	TimeEnd   *time.Time `json:"time_end"`
	Date      time.Time  `json:"date"`
	IsWork    bool       `json:"isWork"`
}

type Patient struct {
	ExternalID int64      `json:"externalID,omitempty"` // Идентификатор из внешней системы
	Name       string     `json:"name"`
	Surname    string     `json:"surname"`
	SecondName *string    `json:"secondName,omitempty"`
	Birthday   *time.Time `json:"birthday,omitempty"`
	Sex        *int       `json:"sex,omitempty"` // Пол (1-мужской, 0-женский, null, если неизвестно)
	Comments   *string    `json:"comments,omitempty"`
	Phone      string     `json:"phone"`
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

func (c *DentalProClient) AvailableAppointments(
	userID int64, doctorIDS []int64, isPlanned bool) (map[int64]map[int64]Appointment, error) {
	// Приемы доступные к записи
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=mobile/records/appointmentsList&target=modal
	return nil, nil
}

func (c *DentalProClient) CreatePatient(name, surname string, phone string) (Patient, error) {
	// Добавление пациента
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/createClient&target=modal
	return Patient{}, nil
}

func (c *DentalProClient) DoctorWorkSchedule(date time.Time, doctorID int64) ([]WorkSchedule, error) {
	// url: /api/mobile/doctorSchedule/doctorMonthGraph
	return nil, nil
}

func (c *DentalProClient) PatientByPhone(phone string) (Patient, error) {
	// Отдает пациента по его номеру телефона
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=client_by_phone&target=modal
	return Patient{}, nil
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
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var responseData []TimesheetResponse
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
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
