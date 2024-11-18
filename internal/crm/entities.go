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

type EditPatientResponse struct {
	ClientID *int64 `json:"clientID"`
	Status   bool   `json:"status"`
	Message  string `json:"message"`
}

type Record struct {
	ID                  int64          `json:"id"`
	Reason              string         `json:"reason"`
	TimeBegin           TimeHMS        `json:"timeBegin"`
	TimeEnd             TimeHMS        `json:"timeEnd"`
	ChairID             int64          `json:"chairId"`
	ClientID            int64          `json:"clientId"`
	Color               string         `json:"color"`
	IsVIP               bool           `json:"isVip"`
	IsChild             bool           `json:"isChild"`
	IsPolis             bool           `json:"isPolis"`
	DoctorID            int64          `json:"doctorId"`
	Date                DateYMD        `json:"date"`
	ClientName          string         `json:"clientName"`
	ClientFromWhere     int            `json:"clientFromWhere"`
	ClientFromWhereName string         `json:"clientFromWhereName"`
	ClientPhone         string         `json:"clientPhone"`
	ClientExternalID    string         `json:"clientExternalId"`
	ClientGroups        map[int]string `json:"clientGroups"`
	IsPrimary           string         `json:"isPrimary"`
	SendRecordSms       bool           `json:"sendRecordSms"`
	Status              interface{}    `json:"status"`
}

type ShortRecord struct {
	ID                 int64          `json:"id"`
	DateStart          DateTimeYMDHMS `json:"dateStart"`
	DateStartTimestamp int64          `json:"dateStartTimestamp"`
	DateEnd            DateTimeYMDHMS `json:"dateEnd"`
	DateEndTimestamp   int64          `json:"dateEndTimestamp"`
	Duration           int            `json:"duration"`
	Name               string         `json:"name"`
	Teeth              string         `json:"teeth"`
	ToothShort         string         `json:"toothShort"`
	DoctorID           int64          `json:"doctorId"`
	DoctorName         string         `json:"doctorName"`
	DoctorGroup        string         `json:"doctorGroup"`
	BranchID           int            `json:"branchId"`
	Total              int            `json:"total"`
}

type ChangeRecord struct {
	ID      int64  `json:"mediline_record_id" description:"ID приема"`
	Status  bool   `json:"status" description:"Статус записи"`
	Message string `json:"message" description:"Сообщение"`
}

type User struct {
	IDClient               string  `json:"id_client"`
	Key                    string  `json:"key"`
	Value                  string  `json:"value"`
	DisplayName            string  `json:"display_name"`
	Name                   string  `json:"name"`
	Surname                string  `json:"surname"`
	SecondName             string  `json:"second_name"`
	Nickname               *string `json:"nickname,omitempty"`
	IDFilePhoto            *string `json:"id_file_photo,omitempty"`
	Birthday               string  `json:"birthday"`
	Country                *string `json:"country,omitempty"`
	State                  *string `json:"state,omitempty"`
	City                   *string `json:"city,omitempty"`
	Postcode               *string `json:"postcode,omitempty"`
	Address                *string `json:"address,omitempty"`
	Street                 *string `json:"street,omitempty"`
	Building               string  `json:"building"`
	Apt                    string  `json:"apt"`
	IsDeleted              *bool   `json:"is_deleted,omitempty"`
	Sex                    string  `json:"sex"`
	Color                  *string `json:"color,omitempty"`
	ParentID               *string `json:"parent_id,omitempty"`
	DateCreate             string  `json:"date_create"`
	INN                    *string `json:"inn,omitempty"`
	DateOfFirstAppointment string  `json:"date_of_first_appointment"`
	DateOfLastAppointment  string  `json:"date_of_last_appointment"`
	Note                   string  `json:"note"`
	ContactInformation     struct {
		MobilePhone string `json:"mobile_phone"`
	} `json:"contact_information"`
}
