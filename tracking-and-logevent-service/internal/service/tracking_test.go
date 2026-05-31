package service

import (
	"testing"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository/mock"
	"github.com/golang/mock/gomock"
)

func TestGetHistory_Failed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockTrackingRepository(ctrl)

	// Simulasi respons sukses dari database saat GetResiHistory dipanggil.
	dummyHistory := &model.TrackingHistory{
		ResiID: "BUBU-123",
		History: []model.TrackingLog{
			{ActivityCode: "TRANSIT", LocationCode: "CGK"},
		},
	}

	mockRepo.EXPECT().GetResiHistory("BUBU-123").Return(dummyHistory, nil).Times(1)

	svc := NewTrackingService(mockRepo)
	result, err := svc.GetHistory("BUBU-123")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Memastikan pengujian gagal karena panjang slice yang dikembalikan adalah nol.
	if len(result.History) == 0 {
		t.Errorf("Test failed! Expected transit history, but got empty slice")
	}
}