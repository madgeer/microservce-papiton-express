//go:build functional

package functional_test

import (
	"testing"
	"warehouse-inventory-service/internal/repository"
	"warehouse-inventory-service/internal/service"
	"warehouse-inventory-service/test/helpers"

	"github.com/stretchr/testify/assert"
)

/*
* Functional Test untuk Arrival Flow.
* Mensimulasikan E2E (End-to-End) flow ketika truk (manifest) akhirnya tiba
* di gudang (warehouse) tujuan pembongkaran.
*/
func TestArrivalFlow_Functional(t *testing.T) {
	// Persiapan Koneksi Database (Setup DB Test)
	db := helpers.SetupTestDB()
	if db != nil {
		defer db.Close()
		defer helpers.CleanTestDB()
	}

	// Persiapan Service & Repository
	var repo repository.ManifestRepository
	if db != nil {
		repo = repository.NewPostgresManifestRepo(db)
	} else {
		t.Fatal("Database test tidak tersedia, functional test gagal")
	}

	svc := service.NewManifestService(repo)

	// Eksekusi Fungsi Skenario Kedatangan
	manifestID, err := repo.CreateManifest("TRK-B6990XYZ", "Sutejo")
	assert.NoError(t, err)
	assert.NotEmpty(t, manifestID)

	destinationWarehouseID := "WH-DESTINATION"

	err = svc.ReceiveManifest(manifestID, destinationWarehouseID)
	assert.NoError(t, err)

	// Verifikasi (Assertion)
	// - Memastikan status manifest berubah menjadi ARRIVED di tabel manifest.
	status, err := repo.GetManifestStatus(manifestID)
	assert.NoError(t, err)
	assert.Equal(t, "ARRIVED", status)
}
