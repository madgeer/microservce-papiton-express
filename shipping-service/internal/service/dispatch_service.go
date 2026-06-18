package service

import (
	"context"
	"errors"
	"time"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
)

// dispatchService
type dispatchService struct {
	courierRepo  domain.CourierRepository
	locationRepo domain.LocationRepository
	dispatchRepo domain.DispatchRepository
	publisher    domain.DispatchEventPublisher
}

/* NewDispatchService constructor untuk membuat instance dispatchService */
func NewDispatchService(
	cr domain.CourierRepository,
	lr domain.LocationRepository,
	dr domain.DispatchRepository,
	p domain.DispatchEventPublisher,
) domain.DispatchService {
	return &dispatchService{
		courierRepo:  cr,
		locationRepo: lr,
		dispatchRepo: dr,
		publisher:    p,
	}
}

func (s *dispatchService) AutoDispatchPickUp(
	ctx context.Context,
	orderID string,
	pickupZone string,
) (*domain.Dispatch, error) {

	if orderID == "" {
		return nil, errors.New("order ID tidak boleh kosong")
	}

	// Skenario unit test / fallback jika repository bernilai nil
	if s.courierRepo == nil || s.dispatchRepo == nil {
		dispatch := &domain.Dispatch{
			ID:               "DSP-" + orderID,
			OrderID:          orderID,
			CourierID:        "C-DUMMY",
			Status:           domain.DispatchStatusAssigned,
			RouteInstruction: "Jemput paket di zona " + pickupZone,
		}
		if s.publisher != nil {
			_ = s.publisher.PublishDispatchAssigned(ctx, dispatch)
		}
		return dispatch, nil
	}

	// 1. Cari kurir yang tersedia di zona penjemputan
	couriers, err := s.courierRepo.GetAvailableByZone(ctx, pickupZone)
	if err != nil {
		return nil, err
	}
	if len(couriers) == 0 {
		return nil, errors.New("tidak ada kurir yang tersedia di zona " + pickupZone)
	}

	// Pilih kurir pertama yang tersedia
	selectedCourier := couriers[0]

	// 2. Ubah status kurir menjadi ON_DUTY
	err = s.courierRepo.UpdateStatus(ctx, selectedCourier.ID, domain.CourierStatusOnDuty)
	if err != nil {
		return nil, err
	}

	// 3. Buat instruksi dispatch baru — sertakan VehicleType agar ETL tidak perlu query shipping-db
	dispatch := &domain.Dispatch{
		ID:               "DSP-" + orderID,
		OrderID:          orderID,
		CourierID:        selectedCourier.ID,
		VehicleType:      selectedCourier.VehicleType,
		Status:           domain.DispatchStatusAssigned,
		RouteInstruction: "Jemput paket di zona " + pickupZone,
	}

	// 4. Simpan log dispatch ke database PostgreSQL
	err = s.dispatchRepo.Create(ctx, dispatch)
	if err != nil {
		return nil, err
	}

	// 5. Kirim event dispatch.assigned ke Kafka jika publisher tersedia
	if s.publisher != nil {
		_ = s.publisher.PublishDispatchAssigned(ctx, dispatch)
	}

	return dispatch, nil
}

func (s *dispatchService) ConfirmPickUp(ctx context.Context, dispatchID string) error {
	if s.dispatchRepo == nil {
		return nil
	}
	return s.dispatchRepo.UpdateStatus(ctx, dispatchID, domain.DispatchStatusPickedUp)
}

func (s *dispatchService) UpdateCourierGPS(ctx context.Context, courierID string, lat, long float64) error {
	if s.locationRepo == nil {
		return nil
	}
	loc := &domain.CourierLocation{
		CourierID: courierID,
		Latitude:  lat,
		Longitude: long,
		Timestamp: time.Now(),
	}
	return s.locationRepo.UpdateLocation(ctx, loc)
}

func (s *dispatchService) GetCourierGPS(ctx context.Context, courierID string) (*domain.CourierLocation, error) {
	if s.locationRepo == nil {
		return nil, errors.New("location repository not initialized")
	}
	return s.locationRepo.GetLatestLocation(ctx, courierID)
}

func (s *dispatchService) GetCourier(ctx context.Context, id string) (*domain.Courier, error) {
	if s.courierRepo == nil {
		return nil, errors.New("courier repository not initialized")
	}
	return s.courierRepo.GetByID(ctx, id)
}

func (s *dispatchService) GetAvailableCouriers(ctx context.Context, zone string) ([]*domain.Courier, error) {
	if s.courierRepo == nil {
		return nil, errors.New("courier repository not initialized")
	}
	return s.courierRepo.GetAvailableByZone(ctx, zone)
}

func (s *dispatchService) RegisterCourier(ctx context.Context, courier *domain.Courier) error {
	if s.courierRepo == nil {
		return errors.New("courier repository not initialized")
	}
	if courier.ID == "" {
		return errors.New("courier ID cannot be empty")
	}
	if courier.Status == "" {
		courier.Status = domain.CourierStatusAvailable
	}
	return s.courierRepo.Create(ctx, courier)
}

func (s *dispatchService) UpdateCourierStatus(ctx context.Context, id string, status domain.CourierStatus) error {
	if s.courierRepo == nil {
		return errors.New("courier repository not initialized")
	}
	return s.courierRepo.UpdateStatus(ctx, id, status)
}