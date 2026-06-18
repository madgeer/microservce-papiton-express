package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"order-tariff-service/internal/domain"
)

// validateOrderRequest memastikan semua field wajib terisi dan nilainya masuk akal.
func validateOrderRequest(req domain.OrderRequest) error {
	if strings.TrimSpace(req.Sender.Name) == "" {
		return errors.New("nama pengirim wajib diisi")
	}
	if strings.TrimSpace(req.Sender.City) == "" {
		return errors.New("kota pengirim wajib diisi")
	}
	if strings.TrimSpace(req.Recipient.Name) == "" {
		return errors.New("nama penerima wajib diisi")
	}
	if strings.TrimSpace(req.Recipient.City) == "" {
		return errors.New("kota penerima wajib diisi")
	}

	validServiceTypes := map[string]bool{
		domain.ServiceTypeExpress: true,
		domain.ServiceTypeRegular: true,
		domain.ServiceTypeKargo:   true,
	}
	if !validServiceTypes[strings.ToUpper(req.ServiceType)] {
		return errors.New("service_type tidak valid, gunakan EXPRESS / REGULAR / CARGO")
	}
	req.ServiceType = strings.ToUpper(req.ServiceType)

	if req.Package.ActualWeight <= 0 {
		return errors.New("berat paket harus lebih dari 0")
	}
	if req.Package.Length <= 0 || req.Package.Width <= 0 || req.Package.Height <= 0 {
		return errors.New("dimensi paket (panjang, lebar, tinggi) harus lebih dari 0")
	}

	lat := req.Sender.Coordinate.Latitude
	lon := req.Sender.Coordinate.Longitude
	if lat != 0 && lon != 0 {
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return errors.New("koordinat pengirim tidak valid")
		}
	}

	lat = req.Recipient.Coordinate.Latitude
	lon = req.Recipient.Coordinate.Longitude
	if lat != 0 && lon != 0 {
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return errors.New("koordinat penerima tidak valid")
		}
	}

	return nil
}

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
	if err := validateOrderRequest(req); err != nil {
		return domain.OrderResponse{}, err
	}
	req.ServiceType = strings.ToUpper(req.ServiceType)

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

	// Publish event ke Kafka jika publisher tersedia.
	// Event diperkaya dengan seluruh data order agar consumer (ETL, Notification)
	// tidak perlu query balik ke database order-service.
	if s.publisher != nil {
		event := domain.OrderCreatedEvent{
			AWB:           response.AWB,
			Email:         req.Sender.Email,
			Status:        response.Status,
			Timestamp:     time.Now(),
			ServiceType:   req.ServiceType,
			HasInsurance:  req.HasInsurance,
			HasPacking:    req.HasPacking,
			SenderCity:    req.Sender.City,
			RecipientCity: req.Recipient.City,
			PackageWeight: req.Package.ActualWeight,
			PackageLength: req.Package.Length,
			PackageWidth:  req.Package.Width,
			PackageHeight: req.Package.Height,
			TarifTotal:    response.TarifTotal,
			DistanceKM:    response.Distance,
		}
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
	if err := validateOrderRequest(req); err != nil {
		return domain.OrderResponse{}, err
	}
	req.ServiceType = strings.ToUpper(req.ServiceType)

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
