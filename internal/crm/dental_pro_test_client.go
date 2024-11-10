package crm

import (
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type DentalProClientTest struct {
	Token     string
	SecretKey string

	mu *sync.Mutex

	Doctors          map[int64]Doctor
	Appointments     map[int64]map[int64]Appointment
	Patients         map[int64]Patient
	AllFreeIntervals []DayInterval
	Records          map[int64][]Record
}

type RequestError struct {
	Code    int
	Message string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

func GetTestFreeIntervals(path string) []DayInterval {
	response := struct {
		BaseResponse
		Data []DayInterval `json:"data"`
	}{}
	parseJSONFile(&response, filepath.Join(path, "/test_data/free_time_intervals.json"))
	return response.Data
}

func GetTestDoctors(path string) map[int64]Doctor {
	response := struct {
		BaseResponse
		Data []Doctor `json:"data"`
	}{}
	parseJSONFile(&response, filepath.Join(path, "/test_data/doctor_list.json"))
	doctors := make(map[int64]Doctor, len(response.Data))
	for _, doctor := range response.Data {
		doctors[doctor.ID] = doctor
	}
	return doctors
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
		12: {
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

func NewDentalProClientTest(token, path, secretKey string) *DentalProClientTest {
	return &DentalProClientTest{
		Token:            token,
		SecretKey:        secretKey,
		Doctors:          GetTestDoctors(path),
		Appointments:     GetTestAppointments(),
		AllFreeIntervals: GetTestFreeIntervals(path),
		Patients:         GetTestPatients(),
		Records:          map[int64][]Record{},
		mu:               &sync.Mutex{},
	}
}

func (c *DentalProClientTest) DoctorsList() ([]Doctor, error) {
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/test?method=mobile/doctor/list&target=modal
	doctors := make([]Doctor, 0, len(c.Doctors))
	for _, doctor := range c.Doctors {
		doctors = append(doctors, doctor)
	}
	sort.Slice(doctors, func(i, j int) bool {
		return doctors[i].ID < doctors[j].ID
	})
	return doctors, nil
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

func (c *DentalProClientTest) RecordCreate(
	date, timeStart, timeEnd time.Time, doctorID, clientID, appointmentID int64, isPlanned bool,
) (*Record, error) {
	// Запись пациента в расписание по автоприему/по ID medical_receptions
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/create&target=modal
	records, ok := c.Records[clientID]
	if !ok {
		records = make([]Record, 0)
	}

	record := Record{
		ID:        rand.Int63(),
		TimeBegin: DateTimeYMDHMS(timeStart),
		TimeEnd:   DateTimeYMDHMS(timeEnd),
		Date:      DateYMD(date),
		ClientID:  clientID,
		DoctorID:  doctorID,
	}

	c.Records[clientID] = append(records, record)

	return &record, nil
}

func (c *DentalProClientTest) joinUserGroups(userGroups map[string]string) string {
	groups := make([]string, 0, len(userGroups))
	for _, group := range userGroups {
		groups = append(groups, group)
	}
	return strings.Join(groups, "")
}

func (c *DentalProClientTest) PatientRecords(clientID int64) ([]ShortRecord, error) {
	// Записи пациента по ID пациента
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=i/client/records&target=modal
	records, ok := c.Records[clientID]
	if !ok {
		records = make([]Record, 0)
	}
	shortRecords := make([]ShortRecord, len(records))

	for i, record := range records {
		startDatetime := mergeToDatetime(time.Time(record.Date), time.Time(record.TimeBegin))
		endDatetime := mergeToDatetime(time.Time(record.Date), time.Time(record.TimeBegin))
		doctor := c.Doctors[record.DoctorID]
		shortRecords[i] = ShortRecord{
			ID:                 record.ID,
			DateStart:          DateTimeYMDHMS(startDatetime),
			DateStartTimestamp: startDatetime.Unix(),
			DateEnd:            DateTimeYMDHMS(endDatetime),
			DateEndTimestamp:   endDatetime.Unix(),
			Duration:           int((endDatetime.Unix() - startDatetime.Unix()) / 60),
			Name:               "Тестовая запись.",
			DoctorID:           record.DoctorID,
			DoctorName:         doctor.FIO,
			DoctorGroup:        c.joinUserGroups(doctor.Departments),
			BranchID:           3,
			Total:              3000,
		}
	}
	return shortRecords, nil
}
