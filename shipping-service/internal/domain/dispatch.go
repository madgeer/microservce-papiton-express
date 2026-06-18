/*
Filename	: dispatch.go
Deskripsi	: Package domain untuk menampung kontrak dan data dari entitas Dispatch.
*/

package domain

import "context"

type DispatchStatus string

const (
	DispatchStatusAssigned DispatchStatus = "ASSIGNED"
	DispatchStatusPickedUp DispatchStatus = "PICKED_UP"
	DispatchStatusInTransit DispatchStatus = "IN_TRANSIT"
	DispatchStatusDelivered DispatchStatus = "DELIVERED"
	DispatchStatusFailed DispatchStatus = "FAILED"
)

type Dispatch struct {
	ID string `json:"id"`
	OrderID string `json:"order_id"`
	CourierID string `json:"courier_id"`
	Status DispatchStatus `json:"status"`
	RouteInstruction string `json:"route_instruction"`
}

// DispatchRepository 
// kontrak untuk menyimpan log penugasan.
type DispatchRepository interface {
	Create(ctx context.Context, dispatch *Dispatch) error
	UpdateStatus(ctx context.Context, id string, status DispatchStatus) error
	GetByOrderID(ctx context.Context, orderID string) (*Dispatch, error)
}

// DispatchService 
// kontrak untuk Logika bisnis yang akan diimplementasikan di layer service.
type DispatchService interface {
	// AutoDispatchPickUp
	//  fungsi Automatic Dispatching untuk penjemputan.
	AutoDispatchPickUp(ctx context.Context, orderID string, pickupZone string) (*Dispatch, error)
	ConfirmPickUp(ctx context.Context, dispatchID string) error
	UpdateCourierGPS(ctx context.Context, courierID string, lat, long float64) error
	GetCourierGPS(ctx context.Context, courierID string) (*CourierLocation, error)
	GetCourier(ctx context.Context, id string) (*Courier, error)
	GetAvailableCouriers(ctx context.Context, zone string) ([]*Courier, error)
	RegisterCourier(ctx context.Context, courier *Courier) error
	UpdateCourierStatus(ctx context.Context, id string, status CourierStatus) error
}

type DispatchEventPublisher interface {
	PublishDispatchAssigned(ctx context.Context, dispatch *Dispatch) error
}