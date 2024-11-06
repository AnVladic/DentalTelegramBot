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

	Doctors          []Doctor
	Schedule         map[int64][]WorkSchedule
	Appointments     map[int64]map[int64]Appointment
	Patients         map[int64]Patient
	AllFreeIntervals []DayInterval
}

type RequestError struct {
	Code    int
	Message string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

func GetTestFreeIntervals() []DayInterval {
	response := struct {
		BaseResponse
		Data []DayInterval `json:"data"`
	}{}
	parseJSONFile(&response, "internal/crm/test_data/free_time_intervals.json")
	return response.Data
}

func GetTestDoctors() []Doctor {
	response := struct {
		BaseResponse
		Data []Doctor `json:"data"`
	}{}
	parseJSONFile(&response, "internal/crm/test_data/doctor_list.json")
	return response.Data
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

func GetTestPatients() map[int64]Patient {
	return map[int64]Patient{}
}

func NewDentalProClientTest(token, secretKey string) *DentalProClientTest {
	return &DentalProClientTest{
		Token:            token,
		SecretKey:        secretKey,
		Doctors:          GetTestDoctors(),
		Schedule:         GetTestSchedule(),
		Appointments:     GetTestAppointments(),
		AllFreeIntervals: GetTestFreeIntervals(),
		Patients:         GetTestPatients(),
		mu:               &sync.Mutex{},
	}
}

func (c *DentalProClientTest) DoctorsList() ([]Doctor, error) {
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/test?method=mobile/doctor/list&target=modal
	return c.Doctors, nil
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

func (c *DentalProClientTest) FreeIntervals(
	startDate, endDate time.Time,
	departmentID, doctorID, branchID int64, duration int,
) ([]DayInterval, error) {
	// Доступные к записи интервалы
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=mobile/records/appointmentsFreeIntervals&target=modal
	var result []DayInterval

	step := duration / 5
	if step == 0 {
		step = 1
	}

	for _, interval := range c.AllFreeIntervals {
		if interval.Date.SubTime(startDate).Seconds() >= 0 && interval.Date.SubTime(endDate).Seconds() <= 0 {
			var filteredSlots []DaySlot

			for _, slot := range interval.Slots {
				if slot.DoctorID == fmt.Sprintf("%d", doctorID) {
					var validTimeRanges []TimeRange

					count := 0
					var mergedInterval TimeRange
					for _, interval := range slot.Time {
						if count == 0 {
							mergedInterval = TimeRange{Begin: interval.Begin}
						}
						count++
						if count == step {
							mergedInterval.End = interval.End
							validTimeRanges = append(validTimeRanges, mergedInterval)
							count = 0
						}
					}
					if len(validTimeRanges) > 0 {
						slot.Time = validTimeRanges
						filteredSlots = append(filteredSlots, slot)
					}
				}
			}

			if len(filteredSlots) > 0 {
				interval.Slots = filteredSlots
				result = append(result, interval)
			}
		}
	}
	return result, nil
}

func (c *DentalProClientTest) EditPatient(patient Patient) (EditPatientResponse, error) {
	// Редактирование базовой информации о пациенте
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/editClient&target=modal
	editPatient, ok := c.Patients[patient.ExternalID]
	if !ok {
		msg := fmt.Sprintf("patient with externalID %d not found", patient.ExternalID)
		return EditPatientResponse{Status: false, Message: msg}, &RequestError{
			http.StatusNotFound,
			msg,
		}
	}
	editPatient.Phone = patient.Phone
	editPatient.Name = patient.Name
	editPatient.Surname = patient.Surname
	return EditPatientResponse{
		ClientID: &editPatient.ExternalID,
		Status:   true,
	}, nil
}
