package crm

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var client IDentalProClient

func LoadEnv() {
	err := godotenv.Load("../../configs/.env")
	if err != nil {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}
}

func TestMain(m *testing.M) {
	LoadEnv()
	client = NewDentalProClient(
		os.Getenv("DENTAL_PRO_TOKEN"), os.Getenv("DENTAL_PRO_SECRET"), false, "")

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestDoctorsList(t *testing.T) {
	doctors, err := client.DoctorsList()
	require.NoError(t, err, "should not return an error")
	require.Greater(t, len(doctors), 0, "should return at least one doctor")
}

func TestAvailableAppointments(t *testing.T) {
	appointments, err := client.AvailableAppointments(-1, []int64{2}, false)
	require.NoError(t, err, "should not return an error")
	require.Greater(t, len(appointments), 0, "should return at least one appointment")
}

func TestClientRecord(t *testing.T) {
	records, err := client.PatientRecords(24)
	require.NoError(t, err, "should not return an error")
	require.Greater(t, len(records), 0, "should return at least one record")
}

func TestFreeIntervals(t *testing.T) {
	intervals, err := client.FreeIntervals(
		time.Date(2024, 10, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 10, 30, 0, 0, 0, 0, time.UTC),
		2,
		2,
		3,
		15,
	)
	require.NoError(t, err, "should not return an error")
	require.Greater(t, len(intervals), 0, "should return at least one free-interval")
}
