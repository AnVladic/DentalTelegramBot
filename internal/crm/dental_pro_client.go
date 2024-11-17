package crm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
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
	DeleteRecord(recordID int64) (ChangeRecord, error)
}

type DentalProClient struct {
	Token          string
	SecretKey      string
	baseURL        string
	client         *http.Client
	last429Request time.Time
	requestMu      sync.Mutex
}

type RequestError struct {
	Code int
	Err  error
}

func (e RequestError) Error() string {
	return fmt.Sprintf("error: %v, status: %d", e.Err, e.Code)
}

func (e RequestError) Unwrap() error {
	return e.Err
}

func NewDentalProClient(token string, secretKey string, test bool, testPath string) IDentalProClient {
	if test {
		return NewDentalProClientTest(token, testPath, secretKey)
	}
	return &DentalProClient{
		Token: token, SecretKey: secretKey, baseURL: "https://olimp.crm3.dental-pro.online/",
		client:         &http.Client{Timeout: 10 * time.Second},
		last429Request: time.Now(),
		requestMu:      sync.Mutex{},
	}
}

func (c *DentalProClient) postRequest(path string, query url.Values, body []byte, data any) error {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	for range 5 {
		duration := time.Until(c.last429Request.Add(3 * time.Second))
		if duration > 0 {
			time.Sleep(duration)
		}

		err := c.tryPostRequest(path, query, body, data)
		if err == nil {
			return nil
		}

		var requestError *RequestError
		if !errors.As(err, &requestError) || requestError.Code != http.StatusTooManyRequests {
			return err
		} else {
			c.last429Request = time.Now()
		}
	}
	return RequestError{Code: http.StatusTooManyRequests, Err: errors.New("too many requests")}
}

func (c *DentalProClient) tryPostRequest(path string, query url.Values, body []byte, data any) error {
	s, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		logrus.Fatal(err)
	}

	req, err := http.NewRequest("POST", s, bytes.NewBuffer(body))
	if err != nil {
		return &RequestError{
			Code: http.StatusInternalServerError,
			Err:  err,
		}
	}
	req.Header.Set("Content-Type", "application/json")
	query.Add("token", c.Token)
	query.Add("secret", c.SecretKey)
	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return &RequestError{
			Code: http.StatusInternalServerError,
			Err:  err,
		}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		logrus.Printf("error: server return status %d", resp.StatusCode)
		return &RequestError{
			Code: resp.StatusCode,
			Err:  fmt.Errorf("error: server return status %d", resp.StatusCode),
		}
	}
	//b, _ := io.ReadAll(resp.Body)
	//
	//// Print the response body
	//fmt.Println("Response Body:", string(b))
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return err
	}
	return nil
}

func (c *User) ToPatient() (*Patient, error) {
	sex := 0
	if c.Sex == "1" {
		sex = 1
	}

	clientID, err := strconv.ParseInt(c.IDClient, 10, 64)
	if err != nil {
		return nil, err
	}

	patient := &Patient{
		ExternalID: clientID,
		Name:       c.Name,
		Surname:    c.Surname,
		SecondName: &c.SecondName,
		Birthday:   nil,
		Sex:        &sex,
		Comments:   &c.Note,
		Phone:      c.ContactInformation.MobilePhone,
	}

	birthday, err := time.Parse("2006-01-02", c.Birthday)
	if err == nil {
		patient.Birthday = &birthday
	}

	return patient, nil
}

func (c *DentalProClient) DoctorsList() ([]Doctor, error) {
	// Список врачей
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=mobile/doctor/list&target=modal
	response := struct {
		BaseResponse
		Data []Doctor `json:"data"`
	}{}
	err := c.postRequest("/api/mobile/doctor/list", url.Values{}, nil, &response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *DentalProClient) AvailableAppointments(
	userID int64, doctorIDS []int64, isPlanned bool) (map[int64]map[int64]Appointment, error) {
	// Приемы доступные к записи
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=mobile/records/appointmentsList&target=modal
	isPlannedNum := 0
	if isPlanned {
		isPlannedNum = 1
	}
	params := url.Values{
		"userID":    []string{strconv.FormatInt(userID, 10)},
		"isPlanned": []string{strconv.Itoa(isPlannedNum)},
	}
	for _, doctorID := range doctorIDS {
		params.Add("doctorIDS[]", strconv.FormatInt(doctorID, 10))
	}
	response := struct {
		BaseResponse
		Data map[string]map[string]Appointment `json:"data"`
	}{}
	err := c.postRequest("/api/mobile/records/appointmentsList", params, nil, &response)
	if err != nil {
		var unmarshalError *json.UnmarshalTypeError
		if errors.As(err, &unmarshalError) {
			return map[int64]map[int64]Appointment{}, nil
		}
		return nil, err
	}
	data := make(map[int64]map[int64]Appointment, len(response.Data))
	for doctorIDStr, appointments := range response.Data {
		doctorID, err := strconv.ParseInt(doctorIDStr, 10, 64) // Base 10, 64-bit integer
		if err != nil {
			logrus.Fatal(err)
			return nil, &RequestError{
				Code: http.StatusUnprocessableEntity,
				Err:  fmt.Errorf("error: cannot convert doctor ID %s to int64", doctorIDStr),
			}
		}
		data[doctorID] = make(map[int64]Appointment, len(appointments))
		for _, appointment := range appointments {
			data[doctorID][appointment.ID] = appointment
		}
	}
	return data, nil
}

func (c *DentalProClient) CreatePatient(name, surname string, phone string) (Patient, error) {
	// Добавление пациента
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/createClient&target=modal
	params := url.Values{
		"name":    []string{name},
		"surname": []string{surname},
		"phone":   []string{normalizePhoneNumber(phone)},
	}
	response := struct {
		BaseResponse
		Data Patient `json:"data"`
	}{}
	err := c.postRequest("/api/records/createClient", params, nil, &response)
	if err != nil {
		return Patient{}, err
	}
	return response.Data, nil
}

func (c *DentalProClient) PatientByPhone(phone string) (Patient, error) {
	// Отдает пациента по его номеру телефона
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=client_by_phone&target=modal
	phone = normalizePhoneNumber(phone)
	params := url.Values{"phone": []string{phone}}
	response := struct {
		BaseResponse
		Data map[string]User `json:"data"`
	}{}
	err := c.postRequest("/api/client_by_phone", params, nil, &response)
	if err != nil {
		var unmarshalError *json.UnmarshalTypeError
		if errors.As(err, &unmarshalError) {
			return Patient{}, &RequestError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("user with phone %s not found", phone),
			}
		}
		return Patient{}, err
	}
	for _, user := range response.Data {
		patient, err := user.ToPatient()
		if err != nil {
			return Patient{}, &RequestError{Code: http.StatusBadGateway, Err: err}
		}
		return *patient, nil
	}
	return Patient{}, &RequestError{
		Code: http.StatusNotFound, Err: fmt.Errorf("user by %s not fount", phone)}
}

func (c *DentalProClient) FreeIntervals(
	startDate, endDate time.Time,
	departmentID, doctorID, branchID int64, duration int,
) ([]DayInterval, error) {
	// Доступные к записи интервалы
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=twin/freetimeintervals&target=modal
	params := url.Values{
		"date_start":    []string{startDate.Format("2006-01-02")},
		"date_end":      []string{endDate.Format("2006-01-02")},
		"department_id": []string{strconv.FormatInt(departmentID, 10)},
		"doctor_id":     []string{strconv.FormatInt(doctorID, 10)},
		"branch_id":     []string{strconv.FormatInt(int64(branchID), 10)},
		"duration":      []string{strconv.Itoa(duration)},
	}
	response := struct {
		BaseResponse
		Data []DayInterval `json:"data"`
	}{}
	err := c.postRequest("/api/twin/freetimeintervals", params, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *DentalProClient) EditPatient(patient Patient) (EditPatientResponse, error) {
	// Редактирование базовой информации о пациенте
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/editClient&target=modal
	params := url.Values{
		"clientID": []string{strconv.FormatInt(patient.ExternalID, 10)},
		"name":     []string{patient.Name},
		"surname":  []string{patient.Surname},
		"phone":    []string{normalizePhoneNumber(patient.Phone)},
	}
	if patient.SecondName != nil {
		params.Add("secondName", patient.Surname)
	}
	if patient.Birthday != nil {
		params.Add("birthday", patient.Birthday.Format("2006-01-02"))
	}
	if patient.Sex != nil {
		params.Add("sex", strconv.Itoa(*patient.Sex))
	}
	if patient.Comments != nil {
		params.Add("comments", *patient.Comments)
	}
	response := struct {
		BaseResponse
		Data EditPatientResponse `json:"data"`
	}{}
	if err := c.postRequest("/api/records/editClient", params, nil, &response); err != nil {
		return EditPatientResponse{}, err
	}
	return response.Data, nil
}

func (c *DentalProClient) RecordCreate(
	data, timeStart, timeEnd time.Time, doctorID, clientID, appointmentID int64, isPlanned bool,
) (*Record, error) {
	// Запись пациента в расписание по автоприему/по ID medical_receptions
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/create&target=modal
	isPlannedStr := "0"
	if isPlanned {
		isPlannedStr = "1"
	}
	params := url.Values{
		"date":           []string{data.Format("2006-01-02")},
		"time_start":     []string{timeStart.Format("15:04:05")},
		"time_end":       []string{timeEnd.Format("15:04:05")},
		"doctor_id":      []string{strconv.FormatInt(doctorID, 10)},
		"client_id":      []string{strconv.FormatInt(clientID, 10)},
		"appointment_id": []string{strconv.FormatInt(appointmentID, 10)},
		"is_planned":     []string{isPlannedStr},
	}
	response := struct {
		BaseResponse
		Data *Record `json:"data"`
	}{}

	if err := c.postRequest("/api/records/create", params, nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *DentalProClient) PatientRecords(clientID int64) ([]ShortRecord, error) {
	// Записи пациента по ID пациента
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=i/client/records&target=modal
	// Duration возвращается в секундах, нужно конвертировать в минуты
	params := url.Values{"client_id": []string{strconv.FormatInt(clientID, 10)}}
	response := struct {
		BaseResponse
		Data []ShortRecord `json:"data"`
	}{}
	err := c.postRequest("/api/i/client/records", params, nil, &response)
	if err != nil {
		return nil, err
	}
	for i := range response.Data {
		response.Data[i].Duration /= 60
	}
	return response.Data, nil
}

func (c *DentalProClient) DeleteRecord(recordID int64) (ChangeRecord, error) {
	// Удаление записи из расписания
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/detail?method=records/deleteMedilineRecord&target=modal
	params := url.Values{"mediline_record_id": []string{strconv.FormatInt(recordID, 10)}}
	response := struct {
		BaseResponse
		Data ChangeRecord `json:"data"`
	}{}
	err := c.postRequest("/api/records/deleteMedilineRecord", params, nil, &response)
	if err != nil {
		return ChangeRecord{}, err
	}
	return response.Data, nil
}

func GetDoctorByID(doctors []Doctor, doctorID int64) *Doctor {
	for _, doctor := range doctors {
		if doctor.ID == doctorID {
			return &doctor
		}
	}
	return nil
}
