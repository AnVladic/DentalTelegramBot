package crm

import "time"

type BaseResponse struct {
	Method     string      `json:"method"`
	Status     bool        `json:"status"`
	Error      interface{} `json:"error"`
	Page       int         `json:"page"`
	Total      int         `json:"total"`
	Paginated  bool        `json:"paginated"`
	Limit      int         `json:"limit"`
	APIVersion int         `json:"api_version"`
	Version    string      `json:"version"`
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
	ID             int64   `json:"id"`
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
	DateAdded    DateTimeYMDHMS    `json:"date_added"`
	DateDelete   *DateTimeYMDHMS   `json:"date_delete"`
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

type TimeRange struct {
	Begin TimeHMS `json:"begin"`
	End   TimeHMS `json:"end"`
}

type DaySlot struct {
	DoctorID   string      `json:"doctor_id"`
	DoctorName string      `json:"doctor_name"`
	Time       []TimeRange `json:"time"`
}

type DayInterval struct {
	Date  DateYMD   `json:"date"`
	Slots []DaySlot `json:"slots"`
}
