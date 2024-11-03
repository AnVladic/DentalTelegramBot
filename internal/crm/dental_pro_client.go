package crm

import (
	"time"
)

type IDentalProClient interface {
	DoctorsList() ([]Doctor, error)
	AvailableAppointments(
		userID int64, doctorIDs []int64, isPlanned bool) (map[int64]map[int64]Appointment, error)

	CreatePatient(name, surname string, phone string) (Patient, error)
	PatientByPhone(phone string) (Patient, error)
	FreeIntervals(
		startDate, endDate time.Time,
		departmentID, doctorID, branchID int64, duration int,
	) ([]DayInterval, error)
}

type DentalProClient struct {
	Token     string
	SecretKey string
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
	// Список врачей
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=mobile/doctor/list&target=modal
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

func (c *DentalProClient) PatientByPhone(phone string) (Patient, error) {
	// Отдает пациента по его номеру телефона
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=client_by_phone&target=modal
	return Patient{}, nil
}

func (c *DentalProClient) FreeIntervals(
	startDate, endDate time.Time,
	departmentID, doctorID, branchID int64, duration int,
) ([]DayInterval, error) {
	// Доступные к записи интервалы
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=twin/freetimeintervals&target=modal
	return []DayInterval{}, nil
}

func GetDoctorByID(doctors []Doctor, doctorID int64) *Doctor {
	for _, doctor := range doctors {
		if doctor.ID == doctorID {
			return &doctor
		}
	}
	return nil
}
