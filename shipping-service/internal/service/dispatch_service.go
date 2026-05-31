package service

import (
	"context"
	"errors"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
)

var ErrNotImplemented = errors.New("not implemented")

// dispatchService
// implementasi dari domain.DispatchService.
type dispatchService struct {
	courierRepo  domain.CourierRepository
	locationRepo domain.LocationRepository
	dispatchRepo domain.DispatchRepository
}

// NewDispatchService 
// constructor untuk membuat instance dispatchService.
func NewDispatchService(
	cr domain.CourierRepository,
	lr domain.LocationRepository,
	dr domain.DispatchRepository,
) domain.DispatchService {
	return &dispatchService{
		courierRepo:  cr,
		locationRepo: lr,
		dispatchRepo: dr,
	}
}

// AutoDispatchPickUp
// fungsi otomatis untuk mencari kurir terdekat/satu zona.
func (s *dispatchService) AutoDispatchPickUp(
    ctx context.Context,
    orderID string,
    pickupZone string,
) (*domain.Dispatch, error) {

    if orderID == "" {
        return nil, errors.New("order ID tidak boleh kosong")
    }

    return nil, ErrNotImplemented
}

// ConfirmPickUp 
// memperbarui status pesanan menjadi "Picked Up" setelah kurir memindai barcode di lokasi.
func (s *dispatchService) ConfirmPickUp(ctx context.Context, dispatchID string) error {
	return ErrNotImplemented
}

// UpdateCourierGPS 
// mencatat koordinat GPS kurir secara real-time ke MongoDB.
func (s *dispatchService) UpdateCourierGPS(ctx context.Context, courierID string, lat, long float64) error {
	return ErrNotImplemented
}