/*
Filename	: location.go
Deskripsi	: Package domain untuk menampung kontrak dan data dari entitas lokasi.
*/

package domain

import (
	"context"
	"time"
)

// CourierLocation
// menyimpan data koordinat kurir secara real-time.
type CourierLocation struct {
	CourierID string    `json:"courier_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

// LocationRepository 
// kontrak untuk akses data koordinat (MongoDB).
type LocationRepository interface {
	// UpdateLocation memperbarui koordinat kurir terkini.
	UpdateLocation(ctx context.Context, loc *CourierLocation) error
	GetLatestLocation(ctx context.Context, courierID string) (*CourierLocation, error)
}