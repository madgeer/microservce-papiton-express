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

func TestHitungTotalTarif_CongestionSurcharge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockOrderRepository(ctrl)
	mockRepo.EXPECT().
		GetPricingFromCache(gomock.Any()).
		Return(5000.0, nil).
		Times(2) // Akan dipanggil 2 kali untuk pengujian macet vs normal

	svc := &orderService{repo: mockRepo}

	// Request ke area Kopo (macet)
	req := domain.OrderRequest{
		Package: domain.Paket{
			Length:       10,
			Width:        10,
			Height:       10,
			ActualWeight: 2.0,
		},
		Recipient: domain.Customer{
			FullAddress: "Jalan Raya Kopo No. 123, Bandung",
			City:        "Bandung",
		},
		ServiceType: "REGULAR",
	}

	hasilMacet := svc.hitungTotalTarif(req, 10.0)

	// Request ke area non-macet
	reqNormal := req
	reqNormal.Recipient.FullAddress = "Jalan Ir. H. Juanda No. 12, Bandung"
	
	hasilNormal := svc.hitungTotalTarif(reqNormal, 10.0)

	// Harusnya hasil ke Kopo lebih mahal karena ada congestion surcharge (+25%)
	assert.Greater(t, hasilMacet, hasilNormal)
	assert.Equal(t, hasilNormal * 1.25, hasilMacet)
}
