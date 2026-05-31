package service

import (
	"order-tariff-service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRESI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockOrderRepository(ctrl)

	// jika input BANDUNG, kemballikan BDG
	mockRepo.EXPECT().GetCityCode("BANDUNG").Return("BDG", nil).Times(1)

	svc := &orderService{repo: mockRepo}
	resi := svc.GenerateAWB("BANDUNG")

	// Ccekek apakah resi mengandung kode kota yang benar
	assert.Contains(t, resi, "BDG")
	assert.NotEmpty(t, resi)
}
