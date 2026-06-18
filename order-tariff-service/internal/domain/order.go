package domain

import (
	"context"
	"time"
)

// konstanta agar mudah dipakai
const (
	ServiceTypeExpress = "EXPRESS"
	ServiceTypeRegular = "REGULAR"
	ServiceTypeKargo   = "CARGO"
)

type Koordinat struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Customer struct {
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	FullAddress string    `json:"full_address"`
	City        string    `json:"city"`
	Coordinate  Koordinat `json:"coordinate"`
}

type Paket struct {
	Length           float64 `json:"length"`
	Width            float64 `json:"width"`
	Height           float64 `json:"height"`
	ActualWeight     float64 `json:"actual_weight"`
	VolumetricWeight float64 `json:"volumetric_weight"`
}

// Order Request dan Response
type OrderRequest struct {
	Sender       Customer `json:"sender"`
	Recipient    Customer `json:"recipient"`
	Package      Paket    `json:"package"`
	ServiceType  string   `json:"service_type"`
	HasInsurance bool     `json:"has_insurance"`
	HasPacking   bool     `json:"has_packing"`
}

type OrderResponse struct {
	AWB        string    `json:"awb"`
	TarifTotal float64   `json:"total"`
	Distance   float64   `json:"distance"`
	ETA        string    `json:"eta"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// OrderCreatedEvent membawa semua data yang dibutuhkan consumer (Notification, ETL)
// agar mereka tidak perlu query balik ke order-db.
type OrderCreatedEvent struct {
	AWB              string    `json:"awb"`
	Email            string    `json:"email"`
	Status           string    `json:"status"`
	Timestamp        time.Time `json:"timestamp"`
	ServiceType      string    `json:"service_type"`
	HasInsurance     bool      `json:"has_insurance"`
	HasPacking       bool      `json:"has_packing"`
	SenderCity       string    `json:"sender_city"`
	RecipientCity    string    `json:"recipient_city"`
	PackageWeight    float64   `json:"package_weight"`
	PackageLength    float64   `json:"package_length"`
	PackageWidth     float64   `json:"package_width"`
	PackageHeight    float64   `json:"package_height"`
	TarifTotal       float64   `json:"tarif_total"`
	DistanceKM       float64   `json:"distance_km"`
}

// INTERFACE
type OrderService interface {
	CreateOrder(req OrderRequest) (OrderResponse, error)
	GetAllOrders() ([]OrderResponse, error)
	GetOrderByAWB(awb string) (*OrderResponse, error)
	CalculateTariff(req OrderRequest) (OrderResponse, error)
}

type OrderRepository interface {
	SaveOrder(req OrderRequest, res OrderResponse) error
	GetDistance(origin, dest Koordinat) (float64, error) //postgis
	GetPricingFromCache(key string) (float64, error)     //redis
	GetCityCode(cityName string) (string, error)         //mengambil kode kota untuk awb
	FindAll() ([]OrderResponse, error)
	FindByAWB(awb string) (*OrderResponse, error)
}

type OrderEventPublisher interface {
	PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error
}
