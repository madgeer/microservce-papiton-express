package service

import (
	"order-tariff-service/internal/domain"
)

type orderService struct {
	repo domain.OrderRepository
}

func NewOrderService(r domain.OrderRepository) domain.OrderService {
	return &orderService{
		repo: r,
	}
}

func (s *orderService) CreateOrder(req domain.OrderRequest) (domain.OrderResponse, error) {
	// mengambil jarak
	dist, err := s.repo.GetDistance(req.Sender.Coordinate, req.Recipient.Coordinate)
	if err != nil {
		dist = 0.0
	}

	//menghitung total tariff
	tarifTotal := s.hitungTotalTarif(req, dist)

	//untuk generate no resi
	awb := s.GenerateAWB(req.Sender.City)

	//Menghitung Estimasi Waktu Sampai
	eta := s.hitungETA(req.ServiceType, dist)

	response := domain.OrderResponse{
		AWB:        awb,
		TarifTotal: tarifTotal,
		Distance:   dist,
		ETA:        eta,
		Status:     "Shipment Created",
	}

	//Simpan ke Database (menggunakan repo)
	err = s.repo.SaveOrder(req, response)
	if err != nil {
		return domain.OrderResponse{}, err
	}

	return response, nil
}
