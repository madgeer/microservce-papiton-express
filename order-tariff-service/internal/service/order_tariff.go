package service

import (
	"order-tariff-service/internal/domain"
)

const (
	PembagiVolumetrik = 6000
	HargaPerKg        = 5000
)

func (s *orderService) hitungTotalTarif(req domain.OrderRequest, dist float64) float64 {
	//logic belum dibuat
	return 0.0
}
