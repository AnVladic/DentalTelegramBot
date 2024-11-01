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

type Doctor struct {
	ID  int64
	FIO string
}
