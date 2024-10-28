package crm

import "time"

type DentalProClientTest struct {
	Token     string
	SecretKey string
}

func (c *DentalProClientTest) Timesheet(startDate, endDate time.Time) ([]TimesheetResponse, error) {
	now := time.Now()
	return []TimesheetResponse{
		{Id: 1, PlannedStart: now, PlannedEnd: now.Add(1 * 24 * time.Hour), ActualStart: startDate.Add(1 * time.Hour), ActualEnd: startDate.Add(1*24*time.Hour + 1*time.Hour), UserID: 123, BranchID: 1},
		{Id: 2, PlannedStart: now.Add(1 * 24 * time.Hour), PlannedEnd: now.Add(2 * 24 * time.Hour), ActualStart: startDate.Add(1*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(2*24*time.Hour + 1*time.Hour), UserID: 124, BranchID: 2},
		{Id: 3, PlannedStart: now.Add(2 * 24 * time.Hour), PlannedEnd: now.Add(3 * 24 * time.Hour), ActualStart: startDate.Add(2*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(3*24*time.Hour + 1*time.Hour), UserID: 125, BranchID: 1},
		{Id: 4, PlannedStart: now.Add(17 * 24 * time.Hour), PlannedEnd: now.Add(4 * 24 * time.Hour), ActualStart: startDate.Add(3*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(4*24*time.Hour + 1*time.Hour), UserID: 126, BranchID: 2},
		{Id: 5, PlannedStart: now.Add(4 * 24 * time.Hour), PlannedEnd: now.Add(5 * 24 * time.Hour), ActualStart: startDate.Add(4*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(5*24*time.Hour + 1*time.Hour), UserID: 127, BranchID: 1},
		{Id: 6, PlannedStart: now.Add(5 * 24 * time.Hour), PlannedEnd: now.Add(6 * 24 * time.Hour), ActualStart: startDate.Add(5*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(6*24*time.Hour + 1*time.Hour), UserID: 128, BranchID: 2},
		{Id: 7, PlannedStart: now.Add(14 * 24 * time.Hour), PlannedEnd: now.Add(7 * 24 * time.Hour), ActualStart: startDate.Add(6*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(7*24*time.Hour + 1*time.Hour), UserID: 129, BranchID: 1},
		{Id: 8, PlannedStart: now.Add(7 * 24 * time.Hour), PlannedEnd: now.Add(8 * 24 * time.Hour), ActualStart: startDate.Add(7*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(8*24*time.Hour + 1*time.Hour), UserID: 130, BranchID: 2},
		{Id: 9, PlannedStart: now.Add(8 * 24 * time.Hour), PlannedEnd: now.Add(9 * 24 * time.Hour), ActualStart: startDate.Add(8*24*time.Hour + 1*time.Hour), ActualEnd: startDate.Add(9*24*time.Hour + 1*time.Hour), UserID: 131, BranchID: 1},
		{Id: 10, PlannedStart: time.Now(), PlannedEnd: time.Now().Add(2 * 24 * time.Hour), ActualStart: time.Now().Add(1 * time.Hour), ActualEnd: time.Now().Add(3 * 24 * time.Hour), UserID: 132, BranchID: 2}, // Пример с текущими датами
	}, nil
}
