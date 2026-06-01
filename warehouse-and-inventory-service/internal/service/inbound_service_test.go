package service

/*
* import package yang dibutuhkan
* testing : package untuk melakukan unit testing
* assert : package untuk melakukan assertion
* gomock : package untuk melakukan mocking dengan GoMock
* mocks : package mock repository yang di-generate oleh mockgen
*/
import (
	"testing"
	"warehouse-inventory-service/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

/*
* unit test untuk InboundProcess
*/
func TestProcessInbound_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := NewInboundService(mockRepo, nil)

	// Contoh resi dengan format: [kode kota]YYMMDDHHMMSS + 4 char random
	// BDO = Bandung, 240430120000 = YYMMDDHHMMSS, X1Y2 = 4 char random
	resi := "BDO240430120000X1Y2"

	// Ekspektasi: Fungsi UpdateStockStatus HARUS dipanggil oleh service
	mockRepo.EXPECT().UpdateStockStatus(resi, "WH-UPI", "AT_HUB").Return(nil)

	// Jalankan fungsi
	err := svc.ProcessInbound(resi, "WH-UPI")

	// ASSERTION: Pasti FAIL karena ProcessInbound di inbound.go belum manggil repo
	assert.NoError(t, err)
}

/*
* unit test untuk ValidatePackage
*/
func TestValidatePackage_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := NewInboundService(mockRepo, nil)

	isValid, isExpress, instructions := svc.ValidatePackage("BDO240430120000X1Y2")
	
	// Secara default return true, false, []
	assert.True(t, isValid)
	assert.False(t, isExpress)
	assert.Empty(t, instructions)
}

/*
* unit test untuk AssignStorageZone. AssignStorageZone itu function untuk menentukan area penyimpanan sementara berdasarkan prioritas paket.
* contohnya paket express di prioritize dan disimpan di ZONE_EXPRESS, sedangkan paket reguler di prioritize dan disimpan di ZONE_REGULAR
*/
func TestAssignStorageZone_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := NewInboundService(mockRepo, nil)

	// Test untuk paket Express
	zoneExpress := svc.AssignStorageZone("BDO240430120000X1Y2", true)
	assert.Equal(t, "ZONE_EXPRESS", zoneExpress)

	// Test untuk paket Reguler
	zoneReguler := svc.AssignStorageZone("BDO240430120000X1Y2", false)
	assert.Equal(t, "ZONE_REGULAR", zoneReguler)
}

/*
* unit test untuk ApplySpecialHandling
*/
func TestApplySpecialHandling_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := NewInboundService(mockRepo, nil)

	// Skenario 1: Tanpa instruksi khusus
	err := svc.ApplySpecialHandling("BDO240430120000X1Y2", []string{})
	assert.NoError(t, err)

	// Skenario 2: Dengan instruksi khusus (misal "FRAGILE")
	// Tambahkan ekspektasi agar test failed (karena implementasi belum memanggil repo)
	mockRepo.EXPECT().UpdatePackageMetadata("BDO240430120000X1Y2", []string{"FRAGILE"}).Return(nil)
	err = svc.ApplySpecialHandling("BDO240430120000X1Y2", []string{"FRAGILE"})
	assert.NoError(t, err)
}