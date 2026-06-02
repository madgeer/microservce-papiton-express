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
* Functional Test untuk Manifest Flow.
* Mensimulasikan E2E (End-to-End) flow: 
* Truk dibuatkan manifest -> Paket dimasukkan -> Disegel (Finalize) -> Truk Berangkat.
*/
func TestManifestFlow_Functional(t *testing.T) {
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

	// Eksekusi Fungsi Skenario Penuh
	truckID := "TRK-B6990XYZ"
	driverName := "Sutejo"
	resi := "BDO240430120000X1Y2"

	// Masukkan dummy package terlebih dahulu karena ada foreign key constraint di tabel manifest_packages
	_, err := db.Exec("INSERT INTO inbound_packages (resi, warehouse_id, status) VALUES ($1, 'WH-UPI', 'AT_HUB')", resi)
	assert.NoError(t, err)

	manifestID, err := svc.CreateNewManifest(truckID, driverName)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifestID)

	err = svc.AddToManifest(manifestID, resi)
	assert.NoError(t, err)

	err = svc.FinalizeManifest(manifestID)
	assert.NoError(t, err)

	err = svc.DepartManifest(manifestID)
	assert.NoError(t, err)

	// Verifikasi (Assertion)
	// Cek apakah status truk di database berubah menjadi DEPARTED (Berangkat)
	status, err := repo.GetManifestStatus(manifestID)
	assert.NoError(t, err)
	assert.Equal(t, "DEPARTED", status)
}
