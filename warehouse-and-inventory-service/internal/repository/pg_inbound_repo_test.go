package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresInboundRepo_UpdateStockStatus_Mock(t *testing.T) {
	// Setup go-sqlmock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Inisialisasi PostgresInboundRepo tapi memasukkan 'db' palsu (mock)
	repo := NewPostgresInboundRepo(db)

	// Kita ekspektasikan fungsi UpdateStockStatus akan mengeksekusi query UPSERT
	mock.ExpectExec("INSERT INTO inbound_packages").
		WithArgs("RESI-001", "WH-UPI", "AT_HUB").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Jalankan fungsinya
	err = repo.UpdateStockStatus("RESI-001", "WH-UPI", "AT_HUB")

	assert.NoError(t, err)
	
	// Verifikasi apakah ekspektasi query dijalankan. 
	// Ini akan error (failed) karena di dalam UpdateStockStatus belum ada kodingan yang mengeksekusi query.
	assert.NoError(t, mock.ExpectationsWereMet(), "Ekspektasi gagal: Query SQL belum dieksekusi oleh repository!")
}
