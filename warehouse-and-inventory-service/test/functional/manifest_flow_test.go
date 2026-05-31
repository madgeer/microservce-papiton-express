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

	manifestID, err := svc.CreateNewManifest(truckID, driverName)
	// Kita expect tidak ada panic, namun service masih mengembalikan string kosong / error nill sementara ini
	// Jadi kita hanya memanggil fungsinya sesuai instruksi.
	_ = err

	err = svc.AddToManifest(manifestID, resi)
	_ = err

	err = svc.FinalizeManifest(manifestID)
	_ = err

	err = svc.DepartManifest(manifestID)
	_ = err

	// Verifikasi (Assertion)
	// Cek apakah status truk di database berubah menjadi DEPARTED (Berangkat)
	assert.Fail(t, "Functional test Manifest Flow gagal: Implementasi DB E2E belum tersedia")
}
