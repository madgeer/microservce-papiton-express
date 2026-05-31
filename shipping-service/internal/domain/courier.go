/*
Filename	: courier.go
Deskripsi	: Package domain untuk menampung kontrak dan data dari entitas kurir.
*/

package domain

import "context"

// Status ketersediaan armada.
type CourierStatus string

// Status kurir :
const (
	CourierStatusAvailable CourierStatus = "AVAILABLE"	// Tersedia
	CourierStatusOnDuty    CourierStatus = "ON_DUTY"	// Dalam Tugas / Pekerjaan
	CourierStatusOffline   CourierStatus = "OFFLINE"	// Offline
)

// Entitas Kurir :
type Courier struct {
	ID          string        `json:"id"` 			// Id kurir
	Name        string        `json:"name"` 		// nama kurir
	PhoneNumber string        `json:"phone_number"` // Nomor Telepon Kurir
	Zone        string        `json:"zone"` 		// Zona wilayah pengiriman 
	Status      CourierStatus `json:"status"` 		// Status kurir
	VehicleType string        `json:"vehicle_type"` // jenis kendaraan
}

// Courier Repository
// Kontrak untuk akses data profil kurir (Relasional).
type CourierRepository interface {
	GetByID(ctx context.Context, id string) (*Courier, error)
	// GetAvailableByZone mencari armada yang tersedia berdasarkan zona.
	GetAvailableByZone(ctx context.Context, zone string) ([]*Courier, error)
	UpdateStatus(ctx context.Context, id string, status CourierStatus) error
}