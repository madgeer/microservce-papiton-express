package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
)

var ErrNotImplemented = errors.New("not implemented")

type courierRepository struct {
	db *sql.DB
}

func NewCourierRepository(db *sql.DB) domain.CourierRepository {
	return &courierRepository{db: db}
}

func (r *courierRepository) GetByID(ctx context.Context, id string) (*domain.Courier, error) {
	return nil, ErrNotImplemented
}

func (r *courierRepository) GetAvailableByZone(ctx context.Context, zone string) ([]*domain.Courier, error) {
	return nil, ErrNotImplemented
}

func (r *courierRepository) UpdateStatus(ctx context.Context, id string, status domain.CourierStatus) error {
	return ErrNotImplemented
}