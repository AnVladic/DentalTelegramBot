package database

import (
	"regexp"
	"strings"
)

func normalizePhone(phone *string) *string {
	if phone == nil {
		return nil
	}

	re := regexp.MustCompile(`\D`)
	newPhone := re.ReplaceAllString(*phone, "")

	if !strings.HasPrefix(newPhone, "+") {
		newPhone = "+" + newPhone
	}

	return &newPhone
}
