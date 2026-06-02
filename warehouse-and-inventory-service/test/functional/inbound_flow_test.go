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
* Functional Test untuk Inbound Flow.
* Berbeda dengan Unit Test, simulasikan alur secara penuh End-to-End
* dari pemanggilan Service hingga koneksi ke Database.
*/
func TestInboundFlow_Functional(t *testing.T) {
	// Persiapan Koneksi Database (Setup DB Test)
	db := helpers.SetupTestDB()
	if db != nil {
		defer db.Close()
		defer helpers.CleanTestDB()
	}

	// Persiapan Service & Repository
	var repo repository.InboundRepository
	if db != nil {
		repo = repository.NewPostgresInboundRepo(db)
	} else {
		t.Fatal("Database test tidak tersedia, functional test gagal")
	}

	svc := service.NewInboundService(repo, nil)

	// Eksekusi Fungsi
	// Simulasi paket baru masuk ke Hub
	resi := "BDO240430120000X1Y2"
	warehouseID := "WH-UPI"
	
	// Masukkan dummy package terlebih dahulu agar query UPDATE berjalan dengan sukses
	_, err := db.Exec("INSERT INTO inbound_packages (resi, warehouse_id, status) VALUES ($1, $2, 'CREATED')", resi, warehouseID)
	assert.NoError(t, err)

	err = svc.ProcessInbound(resi, warehouseID)
	assert.NoError(t, err)

	// Verifikasi (Assertion)
	// Cek apakah data benar-benar tersimpan/berubah di database
	status, err := repo.GetItemByResi(resi)
	assert.NoError(t, err)
	assert.Equal(t, "AT_HUB", status)
}
