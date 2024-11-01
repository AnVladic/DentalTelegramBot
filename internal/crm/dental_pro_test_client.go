package crm

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type DentalProClientTest struct {
	Token     string
	SecretKey string

	mu *sync.Mutex

	Doctors      []Doctor
	Schedule     map[int64][]WorkSchedule
	Appointments map[int64]map[int64]Appointment
	Patients     map[int64]Patient
}

type RequestError struct {
	Code    int
	Message string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

func GetTestDoctors() []Doctor {
	dateAdded, _ := time.Parse("2006-01-02 15:04:05", "2023-09-12 10:22:40")
	photo1 := "/content/upload/web/11.10.2032/42/mvFile_92343245014727.svg"
	return []Doctor{
		{
			ID:           2,
			UserID:       26,
			IsVIP:        false,
			DateAdded:    dateAdded,
			DateDelete:   nil,
			IsHidden:     false,
			MoneyPerHour: 4500,
			Branches: map[string]string{
				"2": "OOO Cтоматологический центр Хотьково",
				"3": "ОЛИМП Софрино",
			},
			Name:       "С",
			Surname:    "Подаева",
			SecondName: "Евгеньевна",
			FIO:        "Подаева С.Е.",
			UserGroups: map[string]string{
				"1":  "Директор",
				"4":  "Администратор",
				"6":  "Врач",
				"9":  "Главврач",
				"10": "Руководитель КЦ",
				"24": "Доступ к базе пациентов",
			},
			Departments: map[string]string{
				"2": "Терапевты",
			},
			Photo: &photo1,
			Phone: "79243540544",
		},
		{
			ID:           3,
			UserID:       27,
			IsVIP:        false,
			DateAdded:    dateAdded,
			DateDelete:   nil,
			IsHidden:     false,
			MoneyPerHour: 8800,
			Branches: map[string]string{
				"2": "OOO Cтоматологический центр Хотьково",
			},
			Name:       "Борис",
			Surname:    "Погосян",
			SecondName: "Камоевич",
			FIO:        "Погосян Б.К.",
			UserGroups: map[string]string{
				"6": "Врач",
			},
			Departments: map[string]string{
				"6": "Ортопеды",
			},
			Photo: nil,
			Phone: "791234296492",
		},
	}
}

func GetTestSchedule() map[int64][]WorkSchedule {
	layoutDate := "2006-01-02"
	layoutTime := "15:04"
	return map[int64][]WorkSchedule{
		2: {
			{TimeStart: parseTime("10:20", layoutTime), TimeEnd: parseTime("19:00", layoutTime), Date: parseDate("2024-11-01", layoutDate), IsWork: true},
			{TimeStart: parseTime("15:00", layoutTime), TimeEnd: parseTime("19:00", layoutTime), Date: parseDate("2024-11-02", layoutDate), IsWork: true},
			{TimeStart: nil, TimeEnd: nil, Date: parseDate("2024-11-03", layoutDate), IsWork: false},
			{TimeStart: parseTime("11:10", layoutTime), TimeEnd: parseTime("19:00", layoutTime), Date: parseDate("2024-11-04", layoutDate), IsWork: true},
			{TimeStart: parseTime("10:00", layoutTime), TimeEnd: parseTime("16:00", layoutTime), Date: parseDate("2024-11-05", layoutDate), IsWork: true},
			{TimeStart: nil, TimeEnd: nil, Date: parseDate("2024-11-06", layoutDate), IsWork: false},
			{TimeStart: nil, TimeEnd: nil, Date: parseDate("2024-11-07", layoutDate), IsWork: false},
			{TimeStart: parseTime("11:10", layoutTime), TimeEnd: parseTime("19:00", layoutTime), Date: parseDate("2024-11-08", layoutDate), IsWork: true},
		},
	}
}

func GetTestAppointments() map[int64]map[int64]Appointment {
	return map[int64]map[int64]Appointment{
		2: {
			41: {
				ID:             41,
				Cost:           0,
				Name:           "Проведение профосмотра терапевта.",
				Time:           15,
				Color:          "rgb(120, 202, 93)",
				DiagnosticType: "Проф. Осмотр",
				IsPlanned:      false,
			},
			25: {
				ID:             25,
				Cost:           0,
				Name:           "Повторная консультация + лечение терапевта.",
				Time:           60,
				Color:          "#0af5f1",
				DiagnosticType: "Лечение",
				IsPlanned:      false,
			},
			86: {
				ID:             86,
				Cost:           0,
				Name:           "Повторная консультация терапевта.",
				Time:           30,
				Color:          "#3a8f3f",
				DiagnosticType: "Консультация",
				IsPlanned:      false,
			},
		},
	}
}

func NewDentalProClientTest(token, secretKey string) *DentalProClientTest {
	return &DentalProClientTest{
		Token:        token,
		SecretKey:    secretKey,
		Doctors:      GetTestDoctors(),
		Schedule:     GetTestSchedule(),
		Appointments: GetTestAppointments(),
		mu:           &sync.Mutex{},
	}
}

func (c *DentalProClientTest) DoctorsList() ([]Doctor, error) {
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/test?method=mobile/doctor/list&target=modal
	return c.Doctors, nil
}

func (c *DentalProClientTest) Timesheet(startDate, endDate time.Time) ([]TimesheetResponse, error) {
	now := time.Now()
	return []TimesheetResponse{
		{ID: 1, PlannedStart: now, PlannedEnd: now.Add(1 * 24 * time.Hour), ActualStart: startDate.Add(1 * time.Hour), ActualEnd: startDate.Add(1*24*time.Hour + 1*time.Hour), UserID: 123, BranchID: 1},
		{ID: 2, PlannedStart: now.Add(1 * 24 * time.Hour), PlannedEnd: now.Add(2 * 24 * time.Hour), ActualStart: startDate.Add(1*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(2*24*time.Hour + 1*time.Hour), UserID: 124, BranchID: 2},
		{ID: 3, PlannedStart: now.Add(2 * 24 * time.Hour), PlannedEnd: now.Add(3 * 24 * time.Hour), ActualStart: startDate.Add(2*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(3*24*time.Hour + 1*time.Hour), UserID: 125, BranchID: 1},
		{ID: 4, PlannedStart: now.Add(17 * 24 * time.Hour), PlannedEnd: now.Add(4 * 24 * time.Hour), ActualStart: startDate.Add(3*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(4*24*time.Hour + 1*time.Hour), UserID: 126, BranchID: 2},
		{ID: 5, PlannedStart: now.Add(4 * 24 * time.Hour), PlannedEnd: now.Add(5 * 24 * time.Hour), ActualStart: startDate.Add(4*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(5*24*time.Hour + 1*time.Hour), UserID: 127, BranchID: 1},
		{ID: 6, PlannedStart: now.Add(5 * 24 * time.Hour), PlannedEnd: now.Add(6 * 24 * time.Hour), ActualStart: startDate.Add(5*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(6*24*time.Hour + 1*time.Hour), UserID: 128, BranchID: 2},
		{ID: 7, PlannedStart: now.Add(14 * 24 * time.Hour), PlannedEnd: now.Add(7 * 24 * time.Hour), ActualStart: startDate.Add(6*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(7*24*time.Hour + 1*time.Hour), UserID: 129, BranchID: 1},
		{ID: 8, PlannedStart: now.Add(7 * 24 * time.Hour), PlannedEnd: now.Add(8 * 24 * time.Hour), ActualStart: startDate.Add(7*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(8*24*time.Hour + 1*time.Hour), UserID: 130, BranchID: 2},
		{ID: 9, PlannedStart: now.Add(8 * 24 * time.Hour), PlannedEnd: now.Add(9 * 24 * time.Hour), ActualStart: startDate.Add(8*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(9*24*time.Hour + 1*time.Hour), UserID: 131, BranchID: 1},
		{ID: 10, PlannedStart: time.Now(), PlannedEnd: time.Now().Add(2 * 24 * time.Hour), ActualStart: time.Now().Add(1 * time.Hour), ActualEnd: time.Now().Add(3 * 24 * time.Hour), UserID: 132, BranchID: 2}, // Пример с текущими датами
	}, nil
}

func (c *DentalProClientTest) DoctorWorkSchedule(date time.Time, doctorID int64) ([]WorkSchedule, error) {
	schedule, exists := c.Schedule[doctorID]
	if !exists {
		return nil, &RequestError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("doctor with ID %d not found", doctorID),
		}
	}

	var result []WorkSchedule
	year, month := date.Year(), date.Month()
	for _, entry := range schedule {
		if entry.Date.Year() == year && entry.Date.Month() == month {
			result = append(result, entry)
		}
	}

	return result, nil
}

func (c *DentalProClientTest) AvailableAppointments(
	userID int64, doctorIDS []int64, isPlanned bool) (map[int64]map[int64]Appointment, error) {

	result := make(map[int64]map[int64]Appointment)

	for _, doctorID := range doctorIDS {
		if appointments, ok := c.Appointments[doctorID]; ok {
			filteredAppointments := make(map[int64]Appointment)

			for appID, appointment := range appointments {
				if appointment.IsPlanned == isPlanned {
					filteredAppointments[int64(appID)] = appointment
				}
			}

			if len(filteredAppointments) > 0 {
				result[doctorID] = filteredAppointments
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no available appointments found for userID: %d", userID)
	}

	return result, nil
}

func (c *DentalProClientTest) CreatePatient(name, surname string, phone string) (Patient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	externalID := int64(len(c.Patients) + 1)

	patient := Patient{
		ExternalID: externalID,
		Name:       name,
		Surname:    surname,
		Phone:      phone,
	}

	c.Patients[externalID] = patient
	return patient, nil
}

func (c *DentalProClientTest) PatientByPhone(phone string) (Patient, error) {
	for _, patient := range c.Patients {
		if patient.Phone == phone {
			return patient, nil
		}
	}
	return Patient{}, &RequestError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("patient with phone %s not found", phone),
	}
}

func parseDate(dateStr, layout string) time.Time {
	date, _ := time.Parse(layout, dateStr)
	return date
}

func parseTime(timeStr, layout string) *time.Time {
	if timeStr == "" {
		return nil
	}
	t, _ := time.Parse(layout, timeStr)
	return &t
}
