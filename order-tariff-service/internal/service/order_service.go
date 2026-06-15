package service

import (
	"context"
	"time"

	"order-tariff-service/internal/domain"
)

type orderService struct {
	repo      domain.OrderRepository
	publisher domain.OrderEventPublisher
}

func NewOrderService(r domain.OrderRepository, p domain.OrderEventPublisher) domain.OrderService {
	return &orderService{
		repo:      r,
		publisher: p,
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

	// Hitung dan simpan volumetric weight ke request payload
	req.Package.VolumetricWeight = (req.Package.Length * req.Package.Width * req.Package.Height) / 6000.0

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

	// Publish event ke Kafka jika publisher tersedia
	if s.publisher != nil {
		event := domain.OrderCreatedEvent{
			AWB:       response.AWB,
			Email:     req.Sender.Email,
			Status:    response.Status,
			Timestamp: time.Now(),
		}
		// Diabaikan error-nya untuk resiliency fallback
		_ = s.publisher.PublishOrderCreated(context.Background(), event)
	}

	return response, nil
}

func (s *orderService) GetAllOrders() ([]domain.OrderResponse, error) {
	return s.repo.FindAll()
}

func (s *orderService) GetOrderByAWB(awb string) (*domain.OrderResponse, error) {
	return s.repo.FindByAWB(awb)
}

func (s *orderService) CalculateTariff(req domain.OrderRequest) (domain.OrderResponse, error) {
	dist, err := s.repo.GetDistance(req.Sender.Coordinate, req.Recipient.Coordinate)
	if err != nil {
		dist = 0.0
	}

	tarifTotal := s.hitungTotalTarif(req, dist)
	eta := s.hitungETA(req.ServiceType, dist)

	return domain.OrderResponse{
		AWB:        "CALC-ONLY",
		TarifTotal: tarifTotal,
		Distance:   dist,
		ETA:        eta,
		Status:     "Calculated",
		CreatedAt:  time.Now(),
	}, nil
}
