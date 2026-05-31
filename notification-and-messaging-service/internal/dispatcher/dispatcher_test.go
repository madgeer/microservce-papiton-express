package dispatcher_test

import (
	"context"
	"errors"
	"testing"

	"papiton/notification-service/internal/dispatcher"
	"papiton/notification-service/internal/model"
	"papiton/notification-service/mocks"

	"github.com/golang/mock/gomock"
)

func TestDispatcher_Dispatch_Email_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmail := mocks.NewMockNotificationProvider(ctrl)
	mockPush := mocks.NewMockNotificationProvider(ctrl)
	mockRepo := mocks.NewMockNotificationRepository(ctrl)

	msg := model.NotificationMessage{
		UserID:  "user-001",
		Channel: model.ChannelEmail,
		Subject: "Pengiriman Gagal",
		Body:    "Paket AWB001 gagal dikirim",
		AWB:     "AWB001",
	}

	// Email HARUS dipanggil 1x
	mockEmail.EXPECT().Send(gomock.Any(), msg).Return(nil).Times(1)
	// Push TIDAK boleh dipanggil
	mockPush.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	// Log HARUS disimpan dengan status sukses
	mockRepo.EXPECT().SaveLog(gomock.Any(), msg, true).Return(nil).Times(1)

	d := dispatcher.NewDispatcher(mockEmail, mockPush, mockRepo)
	err := d.Dispatch(context.Background(), msg)

	if err != nil {
		t.Errorf("tidak seharusnya ada error: %v", err)
	}
}

func TestDispatcher_Dispatch_Push_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmail := mocks.NewMockNotificationProvider(ctrl)
	mockPush := mocks.NewMockNotificationProvider(ctrl)
	mockRepo := mocks.NewMockNotificationRepository(ctrl)

	msg := model.NotificationMessage{
		UserID:  "user-002",
		Channel: model.ChannelPush,
		Body:    "Paket kamu sudah sampai!",
		AWB:     "AWB002",
	}

	mockPush.EXPECT().Send(gomock.Any(), msg).Return(nil).Times(1)
	mockEmail.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().SaveLog(gomock.Any(), msg, true).Return(nil).Times(1)

	d := dispatcher.NewDispatcher(mockEmail, mockPush, mockRepo)
	err := d.Dispatch(context.Background(), msg)

	if err != nil {
		t.Errorf("tidak seharusnya ada error: %v", err)
	}
}

func TestDispatcher_Dispatch_ProviderFails_StillSavesLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmail := mocks.NewMockNotificationProvider(ctrl)
	mockPush := mocks.NewMockNotificationProvider(ctrl)
	mockRepo := mocks.NewMockNotificationRepository(ctrl)

	msg := model.NotificationMessage{
		Channel: model.ChannelPush,
		AWB:     "AWB003",
		UserID:  "user-003",
	}

	providerError := errors.New("push notification server timeout")
	mockPush.EXPECT().Send(gomock.Any(), msg).Return(providerError).Times(1)

	// Bahkan saat gagal, log HARUS tetap tersimpan dengan status false
	mockRepo.EXPECT().SaveLog(gomock.Any(), msg, false).Return(nil).Times(1)

	d := dispatcher.NewDispatcher(mockEmail, mockPush, mockRepo)
	err := d.Dispatch(context.Background(), msg)

	if err == nil {
		t.Error("seharusnya ada error karena provider gagal")
	}
}

func TestDispatcher_Dispatch_UnknownChannel_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmail := mocks.NewMockNotificationProvider(ctrl)
	mockPush := mocks.NewMockNotificationProvider(ctrl)
	mockRepo := mocks.NewMockNotificationRepository(ctrl)

	msg := model.NotificationMessage{
		Channel: "sms", // Channel yang belum didukung
		AWB:     "AWB004",
		UserID:  "user-004",
	}

	// Tidak ada provider yang boleh dipanggil
	mockEmail.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	mockPush.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().SaveLog(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	d := dispatcher.NewDispatcher(mockEmail, mockPush, mockRepo)
	err := d.Dispatch(context.Background(), msg)

	if err == nil {
		t.Error("seharusnya ada error untuk channel tidak dikenal")
	}
}
