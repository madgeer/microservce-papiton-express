package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresManifestRepo_CreateManifest_Mock(t *testing.T) {
	// Setup go-sqlmock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Inisialisasi PostgresManifestRepo dengan DB palsu dari sqlmock
	repo := NewPostgresManifestRepo(db)

	// Kita ekspektasikan fungsi CreateManifest akan menjalankan query INSERT 
	// dan me-return baris (ID manifest yang baru dibuat)
	mock.ExpectQuery("INSERT INTO manifests").
		WithArgs("TRK-123", "Budi").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("MNF-001"))

	// Jalankan fungsinya
	id, err := repo.CreateManifest("TRK-123", "Budi")

	assert.NoError(t, err)
	assert.Equal(t, "MNF-001", id)
	
	// Cek apakah query INSERT tadi benar-benar dijalankan. 
	// Pasti FAILED karena belum ada `db.QueryRow(...)` di dalam file aslinya.
	assert.NoError(t, mock.ExpectationsWereMet(), "Ekspektasi gagal: Query INSERT belum dieksekusi oleh repository!")
}
