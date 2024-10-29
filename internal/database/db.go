package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type User struct {
	ID       int64
	TgUserID int64
	Name     *string
	Lastname *string
	Phone    *string
}

func GetOrCreateUser(db *sql.DB, tgUserId int64) (*User, bool, error) {
	ctx := context.Background()
	var user User
	created := false

	err := db.QueryRowContext(ctx,
		`SELECT id, tg_user_id, name, lastname, phone 
        FROM "User" 
        WHERE tg_user_id = $1`, tgUserId,
	).Scan(&user.ID, &user.TgUserID, &user.Name, &user.Lastname, &user.Phone)

	if errors.Is(err, sql.ErrNoRows) {
		err = db.QueryRowContext(ctx,
			`INSERT INTO "User" (tg_user_id) 
            VALUES ($1) RETURNING Id, tg_user_id, name, lastname, phone`, tgUserId,
		).Scan(&user.ID, &user.TgUserID, &user.Name, &user.Lastname, &user.Phone)
		created = true
		if err != nil {
			return nil, created, fmt.Errorf("error creating user: %v", err)
		}
	} else if err != nil {
		return nil, created, fmt.Errorf("error querying user: %v", err)
	}

	return &user, created, nil
}
