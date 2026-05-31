package service

import (
	"order-tariff-service/internal/domain"
	"order-tariff-service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHitungTotalTarif_WithCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockOrderRepository(ctrl)
	mockRepo.EXPECT().
		GetPricingFromCache(gomock.Any()).
		Return(5000.0, nil).
		Times(1)

	svc := &orderService{repo: mockRepo}

	req := domain.OrderRequest{
		Package: domain.Paket{
			Length:       10,
			Width:        10,
			Height:       10,
			ActualWeight: 2.0,
		},
		ServiceType: "REGULAR",
	}

	hasil := svc.hitungTotalTarif(req, 10.0)

	assert.Greater(t, hasil, 0.0)
}
