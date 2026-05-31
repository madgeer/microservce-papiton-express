package service

import (
	"testing"
	"warehouse-inventory-service/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

/*
* Kumpulan Unit Test untuk Sorting Service.
* Ekspektasi dibuat, tapi fungsi asli dikosongkan agar FAILED.
*/

/* TestAssignPackageToLane_Unit untuk testing assign lane */
func TestAssignPackageToLane_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSortingRepository(ctrl)
	svc := NewSortingService(mockRepo)

	// Ekspektasi: Service HARUS menyimpan data ke DB (Mock)
	mockRepo.EXPECT().AssignToLane("BDO240430120000X1Y2", "LANE-JAKARTA").Return(nil)

	err := svc.AssignPackageToLane("BDO240430120000X1Y2", "LANE-JAKARTA")

	assert.NoError(t, err)
}

/* TestGetPackagesInLane_Unit untuk testing get packages in lane */
func TestGetPackagesInLane_Unit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSortingRepository(ctrl)
	svc := NewSortingService(mockRepo)

	// Persiapan data palsu
	dummyResiList := []string{"RESI-01", "RESI-02"}

	// Ekspektasi: Service mengambil data dari DB
	mockRepo.EXPECT().GetPackagesInLane("LANE-JAKARTA").Return(dummyResiList, nil)

	pkgs, err := svc.GetPackagesInLane("LANE-JAKARTA")

	assert.NoError(t, err)
	assert.Len(t, pkgs, 2)
}
