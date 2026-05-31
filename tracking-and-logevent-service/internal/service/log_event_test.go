package service

import (
	"testing"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository/mock"
	"github.com/golang/mock/gomock"
)

func TestProcessLog_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLogEventRepository(ctrl)
	dummyLog := model.TrackingLog{ResiID: "BUBU-123", ActivityCode: "DELIVERED"}

	// Simulasi respons sukses dari database saat InsertLog dipanggil.
	mockRepo.EXPECT().InsertLog(dummyLog).Return(nil).Times(1)

	svc := NewLogEventService(mockRepo)
	err := svc.ProcessLog(dummyLog)

	// Memastikan pengujian gagal karena fungsi ProcessLog mengembalikan error buatan.
	if err != nil {
		t.Errorf("Test failed! Expected success, but got error: %v", err)
	}
}