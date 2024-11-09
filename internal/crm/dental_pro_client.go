package crm

import (
	"time"
)

type IDentalProClient interface {
	DoctorsList() ([]Doctor, error)
	AvailableAppointments(
		userID int64, doctorIDs []int64, isPlanned bool) (map[int64]map[int64]Appointment, error)

	CreatePatient(name, surname string, phone string) (Patient, error)
	EditPatient(patient Patient) (EditPatientResponse, error)

	PatientByPhone(phone string) (Patient, error)
	FreeIntervals(
		startDate, endDate time.Time,
		departmentID, doctorID, branchID int64, duration int,
	) ([]DayInterval, error)
	RecordCreate(
		date, timeStart, timeEnd time.Time, doctorID, clientID, appointmentID int64, isPlanned bool,
	) (*Record, error)
	PatientRecords(clientID int64) ([]ShortRecord, error)
}

type DentalProClient struct {
	Token     string
	SecretKey string
	baseURL   string
}

func NewDentalProClient(token string, secretKey string, test bool, testPath string) IDentalProClient {
	if test {
		return NewDentalProClientTest(token, testPath, secretKey)
	}
	return &DentalProClient{Token: token, SecretKey: secretKey, baseURL: "https://api.dentaltelegram.com"}
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

func (c *DentalProClient) EditPatient(patient Patient) (EditPatientResponse, error) {
	// Редактирование базовой информации о пациенте
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/editClient&target=modal
	return EditPatientResponse{}, nil
}

func (c *DentalProClient) RecordCreate(
	data, timeStart, timeEnd time.Time, doctorID, clientID, appointmentID int64, isPlanned bool,
) (*Record, error) {
	// Запись пациента в расписание по автоприему/по ID medical_receptions
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/create&target=modal
	return nil, nil
}

func (c *DentalProClient) PatientRecords(clientID int64) ([]ShortRecord, error) {
	// Записи пациента по ID пациента
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=i/client/records&target=modal
	// Duration возвращается в секундах, нужно конвертировать в минуты
	return nil, nil
}

func GetDoctorByID(doctors []Doctor, doctorID int64) *Doctor {
	for _, doctor := range doctors {
		if doctor.ID == doctorID {
			return &doctor
		}
	}
	return nil
}
