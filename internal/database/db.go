package database

import "time"

type User struct {
	ID          int64
	TgUserID    int64
	DentalProID *int64
	CreatedAt   time.Time
	Name        *string
	Lastname    *string
	Phone       *string
}

type Register struct {
	ID            int64      `db:"id"`
	UserID        int64      `db:"user_id"`
	MessageID     int        `db:"message_id"`
	ChatID        int64      `db:"chat_id"`
	DoctorID      *int64     `db:"doctor_id"`
	AppointmentID *int64     `db:"appointment_id"`
	Datetime      *time.Time `db:"datetime"`
}

type Doctor struct {
	ID  int64
	FIO string
}
