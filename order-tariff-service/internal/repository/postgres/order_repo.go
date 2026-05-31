package postgres

import (
	"database/sql"
	"order-tariff-service/internal/domain"
)

// struct implementasi repository
type OrderRepositoryImpl struct {
	db *sql.DB
}

// constructor
func NewOrderRepository(db *sql.DB) *OrderRepositoryImpl {
	return &OrderRepositoryImpl{db: db}
}

func (r *OrderRepositoryImpl) SaveOrder(req domain.OrderRequest, res domain.OrderResponse) error {
	// belum implement
	return nil
}

func (r *OrderRepositoryImpl) GetDistance(origin, dest domain.Koordinat) (float64, error) {
	// PostGIS belum dibuat
	return 0.0, nil
}

func (r *OrderRepositoryImpl) GetPricingFromCache(key string) (float64, error) {
	return 0.0, nil
}

func (r *OrderRepositoryImpl) GetCityCode(cityName string) (string, error) {
	return "BDG", nil
}
