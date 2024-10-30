package database

import "time"

type User struct {
	ID        int64
	TgUserID  int64
	CreatedAt time.Time
	Name      *string
	Lastname  *string
	Phone     *string
}
