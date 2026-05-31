package mongo

import (
	"context"
	"errors"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrNotImplemented = errors.New("not implemented")

type locationRepository struct {
	collection *mongo.Collection
}

func NewLocationRepository(db *mongo.Database) domain.LocationRepository {
	return &locationRepository{
		collection: db.Collection("courier_locations"),
	}
}

func (r *locationRepository) UpdateLocation(ctx context.Context, loc *domain.CourierLocation) error {
	return ErrNotImplemented
}

func (r *locationRepository) GetLatestLocation(ctx context.Context, courierID string) (*domain.CourierLocation, error) {
	return nil, ErrNotImplemented
}