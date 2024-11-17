package bot

import (
	"github.com/AnVladic/DentalTelegramBot/internal/crm"
	"github.com/AnVladic/DentalTelegramBot/internal/database"
)

type SelfUser struct {
	tgUser        *database.User
	dentalProUser *crm.Patient
}

func (u *SelfUser) GetSelfFirstName() string {
	if u.dentalProUser != nil && u.dentalProUser.Name != "" {
		return u.dentalProUser.Name
	}
	if u.tgUser != nil && u.tgUser.Name != nil {
		return *u.tgUser.Name
	}
	return ""
}

func (u *SelfUser) GetSelfLastName() string {
	if u.dentalProUser != nil && u.dentalProUser.Surname != "" {
		return u.dentalProUser.Surname
	}
	if u.tgUser != nil && u.tgUser.Lastname != nil {
		return *u.tgUser.Lastname
	}
	return ""
}
