package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresSortingRepo_AssignToLane_Mock(t *testing.T) {
	// Setup go-sqlmock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Inisialisasi PostgresSortingRepo dengan DB palsu dari sqlmock
	repo := NewPostgresSortingRepo(db)

	// Kita ekspektasikan fungsi AssignToLane akan mengeksekusi query UPDATE/INSERT
	mock.ExpectExec("UPDATE inbound_packages SET lane_id = \\$1 WHERE resi = \\$2").
		WithArgs("LANE-01", "RESI-001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Jalankan fungsinya
	err = repo.AssignToLane("RESI-001", "LANE-01")

	assert.NoError(t, err)
	
	// Verifikasi ekspektasi query
	assert.NoError(t, mock.ExpectationsWereMet(), "Ekspektasi gagal: Query SQL belum dieksekusi oleh repository!")
}
