package crm

import "time"

type DentalProClientTest struct {
	Token     string
	SecretKey string
	Doctors   []Doctor
}

func NewDentalProClientTest(token, secretKey string) *DentalProClientTest {
	dateAdded, _ := time.Parse("2006-01-02 15:04:05", "2023-09-12 10:22:40")
	photo1 := "/content/upload/web/11.10.2032/42/mvFile_92343245014727.svg"
	doctors := []Doctor{
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

	return &DentalProClientTest{Token: token, SecretKey: secretKey, Doctors: doctors}
}

func (c *DentalProClientTest) DoctorsList() ([]Doctor, error) {
	// https://olimp.crm3.dental-pro.online/apisettings/api/index#/apisettings/api/test?method=mobile/doctor/list&target=modal
	return c.Doctors, nil
}

func (c *DentalProClientTest) Timesheet(startDate, endDate time.Time) ([]TimesheetResponse, error) {
	now := time.Now()
	return []TimesheetResponse{
		{Id: 1, PlannedStart: now, PlannedEnd: now.Add(1 * 24 * time.Hour), ActualStart: startDate.Add(1 * time.Hour), ActualEnd: startDate.Add(1*24*time.Hour + 1*time.Hour), UserID: 123, BranchID: 1},
		{Id: 2, PlannedStart: now.Add(1 * 24 * time.Hour), PlannedEnd: now.Add(2 * 24 * time.Hour), ActualStart: startDate.Add(1*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(2*24*time.Hour + 1*time.Hour), UserID: 124, BranchID: 2},
		{Id: 3, PlannedStart: now.Add(2 * 24 * time.Hour), PlannedEnd: now.Add(3 * 24 * time.Hour), ActualStart: startDate.Add(2*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(3*24*time.Hour + 1*time.Hour), UserID: 125, BranchID: 1},
		{Id: 4, PlannedStart: now.Add(17 * 24 * time.Hour), PlannedEnd: now.Add(4 * 24 * time.Hour), ActualStart: startDate.Add(3*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(4*24*time.Hour + 1*time.Hour), UserID: 126, BranchID: 2},
		{Id: 5, PlannedStart: now.Add(4 * 24 * time.Hour), PlannedEnd: now.Add(5 * 24 * time.Hour), ActualStart: startDate.Add(4*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(5*24*time.Hour + 1*time.Hour), UserID: 127, BranchID: 1},
		{Id: 6, PlannedStart: now.Add(5 * 24 * time.Hour), PlannedEnd: now.Add(6 * 24 * time.Hour), ActualStart: startDate.Add(5*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(6*24*time.Hour + 1*time.Hour), UserID: 128, BranchID: 2},
		{Id: 7, PlannedStart: now.Add(14 * 24 * time.Hour), PlannedEnd: now.Add(7 * 24 * time.Hour), ActualStart: startDate.Add(6*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(7*24*time.Hour + 1*time.Hour), UserID: 129, BranchID: 1},
		{Id: 8, PlannedStart: now.Add(7 * 24 * time.Hour), PlannedEnd: now.Add(8 * 24 * time.Hour), ActualStart: startDate.Add(7*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(8*24*time.Hour + 1*time.Hour), UserID: 130, BranchID: 2},
		{Id: 9, PlannedStart: now.Add(8 * 24 * time.Hour), PlannedEnd: now.Add(9 * 24 * time.Hour), ActualStart: startDate.Add(8*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(9*24*time.Hour + 1*time.Hour), UserID: 131, BranchID: 1},
		{Id: 10, PlannedStart: time.Now(), PlannedEnd: time.Now().Add(2 * 24 * time.Hour), ActualStart: time.Now().Add(1 * time.Hour), ActualEnd: time.Now().Add(3 * 24 * time.Hour), UserID: 132, BranchID: 2}, // Пример с текущими датами
	}, nil
}
