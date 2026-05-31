package service

import (
	"testing"
	"warehouse-inventory-service/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

/* Kumpulan Unit Test untuk Manifest Service
* Semua tes di bawah ini diset EXPECTATION-nya memanggil DB (Mock).
* Karena service aslinya dikosongkan, tes ini AKAN FAILED.
*/

/* TestCreateNewManifest_Unit untuk testing pembuatan manifest baru */
func TestCreateNewManifest_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockManifestRepository(ctrl)
	svc := NewManifestService(mockRepo)

	// Ekspektasi: Service HARUS memanggil fungsi CreateManifest di Repository
	mockRepo.EXPECT().CreateManifest("TRK-001", "Budi").Return("MNF-123", nil)

	manifestID, err := svc.CreateNewManifest("TRK-001", "Budi")

	assert.NoError(t, err)
	assert.Equal(t, "MNF-123", manifestID)
}

/* TestAddToManifest_Unit untuk testing penambahan paket ke manifest */
func TestAddToManifest_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockManifestRepository(ctrl)
	svc := NewManifestService(mockRepo)

	mockRepo.EXPECT().AddPackageToManifest("MNF-123", "RESI-001").Return(nil)

	err := svc.AddToManifest("MNF-123", "RESI-001")

	assert.NoError(t, err)
}

/* TestFinalizeManifest_Unit untuk testing finalisasi manifest */
func TestFinalizeManifest_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockManifestRepository(ctrl)
	svc := NewManifestService(mockRepo)

	mockRepo.EXPECT().UpdateManifestStatus("MNF-123", "FINALIZED").Return(nil)

	err := svc.FinalizeManifest("MNF-123")

	assert.NoError(t, err)
}

/* TestDepartManifest_Unit untuk testing departure manifest */
func TestDepartManifest_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockManifestRepository(ctrl)
	svc := NewManifestService(mockRepo)

	mockRepo.EXPECT().UpdateManifestStatus("MNF-123", "DEPARTED").Return(nil)

	err := svc.DepartManifest("MNF-123")

	assert.NoError(t, err)
}

/* TestReceiveManifest_Unit untuk testing penerimaan manifest */
func TestReceiveManifest_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockManifestRepository(ctrl)
	svc := NewManifestService(mockRepo)

	mockRepo.EXPECT().UpdateManifestStatus("MNF-123", "ARRIVED").Return(nil)

	err := svc.ReceiveManifest("MNF-123", "WH-DESTINATION")

	assert.NoError(t, err)
}
