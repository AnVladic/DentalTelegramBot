package database

import (
	"database/sql"
)

type UserRepository struct {
	Db *sql.DB
}

func (r *UserRepository) CreateUser(user *User) error {
	query := `
        INSERT INTO "User" (tg_user_id, name, lastname, phone)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `
	err := r.Db.QueryRow(query, user.TgUserID, user.Name, user.Lastname, user.Phone).Scan(&user.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetUserByTelegramID(tgUserID int64) (*User, error) {
	query := `
        SELECT id, tg_user_id, name, lastname, phone, created_at
        FROM "User"
        WHERE tg_user_id = $1;
    `
	user := &User{}
	err := r.Db.QueryRow(query, tgUserID).Scan(
		&user.ID, &user.TgUserID, &user.Name, &user.Lastname, &user.Phone, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpsertUserPhoneByTelegramID(tgUserID int64, phone string) error {
	query := `
        INSERT INTO "User" (tg_user_id, phone)
        VALUES ($1, $2)
        ON CONFLICT (tg_user_id)
        DO UPDATE SET phone = EXCLUDED.phone;
    `

	_, err := r.Db.Exec(query, tgUserID, phone)
	return err
}
