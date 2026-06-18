package service

import (
	"order-tariff-service/internal/domain"
	"order-tariff-service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateOrder_FullFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockOrderRepository(ctrl)

	// mock semua panggilan repo yang ada di CreateOrder
	mockRepo.EXPECT().
		GetDistance(gomock.Any(), gomock.Any()).
		Return(10.0, nil).
		AnyTimes()

	mockRepo.EXPECT().
		SaveOrder(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	mockRepo.EXPECT().
		GetPricingFromCache(gomock.Any()).
		Return(0.0, nil).
		AnyTimes()

	mockRepo.EXPECT().
		GetCityCode(gomock.Any()).
		Return("BDG", nil).
		AnyTimes()

	svc := NewOrderService(mockRepo, nil)
	req := domain.OrderRequest{
		Sender: domain.Customer{
			Name: "Shidqi",
			City: "Bandung",
		},
		Recipient: domain.Customer{
			Name: "Dicka",
			City: "Jakarta",
		},
		Package: domain.Paket{
			ActualWeight: 2.5,
			Length:       10.0,
			Width:        10.0,
			Height:       10.0,
		},
		ServiceType: "REGULAR",
	}

	res, err := svc.CreateOrder(req)

	assert.NoError(t, err)

	assert.NotEmpty(t, res.AWB, "AWB tidak boleh kosong")
	assert.NotEmpty(t, res.ETA, "ETA tidak boleh kosong")

	assert.True(t, res.TarifTotal > 0, "Tarif tidak boleh 0")
}
