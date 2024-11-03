package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type UserRepository struct {
	DB *sql.DB
}

type RegisterRepository struct {
	DB *sql.DB
}

type DoctorRepository struct {
	DB *sql.DB
}

func (r *UserRepository) CreateUser(user *User) error {
	query := `
        INSERT INTO "User" (tg_user_id, name, lastname, phone)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `
	err := r.DB.QueryRow(query, user.TgUserID, user.Name, user.Lastname, user.Phone).Scan(&user.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetUserByTelegramID(tgUserID int64) (*User, error) {
	query := `
        SELECT id, tg_user_id, dental_pro_id, name, lastname, phone, created_at
        FROM "User"
        WHERE tg_user_id = $1;
    `
	user := &User{}
	err := r.DB.QueryRow(query, tgUserID).Scan(
		&user.ID, &user.TgUserID, &user.DentalProID, &user.Name, &user.Lastname, &user.Phone, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetOrCreateByTelegramID(user User) (*User, bool, error) {
	oldUser, err := r.GetUserByTelegramID(user.TgUserID)
	if errors.Is(err, sql.ErrNoRows) {
		err := r.CreateUser(&user)
		if err != nil {
			return nil, false, err
		}
		return &user, true, nil
	}
	return oldUser, false, nil
}

func (r *UserRepository) UpsertPhoneByTelegramID(tgUserID int64, phone string) error {
	query := `
        INSERT INTO "User" (tg_user_id, phone)
        VALUES ($1, $2)
        ON CONFLICT (tg_user_id)
        DO UPDATE SET phone = EXCLUDED.phone;
    `

	_, err := r.DB.Exec(query, tgUserID, phone)
	return err
}

func (r *UserRepository) UpdateDentalProIDByTelegramID(tgUserID int64, dentalProID int64) error {
	query := `
        UPDATE "User"
		SET dental_pro_id = ($1)
		WHERE tg_user_id = ($2);
    `

	_, err := r.DB.Exec(query, dentalProID, tgUserID)
	return err
}

func (r *RegisterRepository) ScanAll(row *sql.Row, register *Register) error {
	return row.Scan(
		&register.ID, &register.UserID, &register.MessageID, &register.ChatID,
		&register.DoctorID, &register.AppointmentID, &register.Datetime,
	)
}

func (r *RegisterRepository) Get(userID int64, chatID int64, messageID int) (*Register, error) {
	query := `
        SELECT id, user_id, message_id, chat_id, doctor_id, appointment_id, datetime
        FROM "Register"
        WHERE user_id = $1 and chat_id = $2 and message_id = $3;
    `
	register := &Register{}
	err := r.ScanAll(r.DB.QueryRow(query, userID, chatID, messageID), register)
	if err != nil {
		return nil, err
	}
	return register, nil
}

func (r *RegisterRepository) Create(register *Register) error {
	query := `
        INSERT INTO "Register" (user_id, message_id, chat_id, doctor_id, appointment_id, datetime)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `
	err := r.DB.QueryRow(query, register.UserID, register.MessageID, register.ChatID,
		register.DoctorID, register.AppointmentID, register.Datetime).Scan(&register.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegisterRepository) GetOrCreate(register Register) (*Register, bool, error) {
	oldRegister, err := r.Get(register.UserID, register.ChatID, register.MessageID)
	if errors.Is(err, sql.ErrNoRows) {
		err := r.Create(&register)
		if err != nil {
			return nil, false, err
		}
		return &register, true, nil
	}
	return oldRegister, false, nil
}

func (r *RegisterRepository) UpsertDoctorID(register Register) (*Register, error) {
	query := `
        INSERT INTO "Register" (user_id, message_id, chat_id, doctor_id, appointment_id, datetime)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id, message_id, chat_id) DO UPDATE
        SET doctor_id = EXCLUDED.doctor_id
        RETURNING id, user_id, message_id, chat_id, doctor_id, appointment_id, datetime;
    `

	updatedRegister := &Register{}
	err := r.ScanAll(r.DB.QueryRow(query,
		register.UserID, register.MessageID, register.ChatID,
		register.DoctorID, register.AppointmentID, register.Datetime),
		updatedRegister)
	if err != nil {
		return &Register{}, fmt.Errorf("failed to upsert register: %w", err)
	}

	return updatedRegister, nil
}

func (r *RegisterRepository) UpdateAppointmentID(register Register) error {
	query := `
        UPDATE "Register"
		SET appointment_id = ($1)
		WHERE user_id = ($2) and chat_id = ($3) and message_id = ($4);
    `

	_, err := r.DB.Exec(query, register.AppointmentID, register.UserID, register.ChatID, register.MessageID)
	return err
}

func (r *RegisterRepository) UpdateDatetime(register Register) error {
	query := `
        UPDATE "Register"
		SET datetime = ($1)
		WHERE user_id = ($2) and chat_id = ($3) and message_id = ($4);
    `

	_, err := r.DB.Exec(query, register.Datetime, register.UserID, register.ChatID, register.MessageID)
	return err
}

func (r *DoctorRepository) Get(id int64) (*Doctor, error) {
	query := `
        SELECT id, fio
        FROM "Doctor"
        WHERE id = $1;
    `
	doctor := &Doctor{}
	err := r.DB.QueryRow(query, id).Scan(&doctor.ID, &doctor.FIO)
	if err != nil {
		return nil, err
	}
	return doctor, nil
}

func (r *DoctorRepository) Create(doctor *Doctor) error {
	query := `
        INSERT INTO "Doctor" (id, fio)
        VALUES ($1, $2)
        RETURNING id;
    `
	err := r.DB.QueryRow(query, doctor.ID, doctor.FIO).Scan(&doctor.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *DoctorRepository) GetOrCreate(doctor Doctor) (*Doctor, bool, error) {
	oldRegister, err := r.Get(doctor.ID)
	if errors.Is(err, sql.ErrNoRows) {
		err := r.Create(&doctor)
		if err != nil {
			return nil, false, err
		}
		return &doctor, true, nil
	}
	return oldRegister, false, nil
}

func (r *DoctorRepository) Upsert(doctor Doctor) error {
	query := `
        INSERT INTO "Doctor" (id, fio)
        VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE
        SET fio = EXCLUDED.fio;
    `
	_, err := r.DB.Exec(query, doctor.ID, doctor.FIO)
	if err != nil {
		return fmt.Errorf("failed to upsert doctor: %w", err)
	}
	return nil
}
