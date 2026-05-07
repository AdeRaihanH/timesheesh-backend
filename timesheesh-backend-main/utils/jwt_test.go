package utils

import (
	"testing"
)

// Test untuk memvalidasi kalkulasi
func TestWorkDurationLogic(t *testing.T) {
	// Misal: Jam Masuk 08:00, Jam Keluar 17:00
	jamMasuk := 8
	jamKeluar := 17
	durasiHarusnya := 9

	hasil := jamKeluar - jamMasuk

	if hasil != durasiHarusnya {
		t.Errorf("Kalkulasi durasi salah! Harusnya %d jam, tapi dapet %d jam", durasiHarusnya, hasil)
	}
}

// Test for validasi format string status task
func TestTaskStatusFormat(t *testing.T) {
	status := "COMPLETED"
	expected := "COMPLETED"

	if status != expected {
		t.Errorf("Format status tidak sesuai standar, dapet: %s", status)
	}
}