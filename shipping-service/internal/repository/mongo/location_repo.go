package mongo

import (
	"context"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type locationRepository struct {
	collection *mongo.Collection
}

/* NewLocationRepository membuat instance baru untuk LocationRepository menggunakan MongoDB */
func NewLocationRepository(db *mongo.Database) domain.LocationRepository {
	return &locationRepository{
		collection: db.Collection("courier_locations"),
	}
}

func (r *locationRepository) UpdateLocation(ctx context.Context, loc *domain.CourierLocation) error {
	filter := bson.M{"courier_id": loc.CourierID}
	update := bson.M{
		"$set": bson.M{
			"latitude":  loc.Latitude,
			"longitude": loc.Longitude,
			"timestamp": loc.Timestamp,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *locationRepository) GetLatestLocation(ctx context.Context, courierID string) (*domain.CourierLocation, error) {
	filter := bson.M{"courier_id": courierID}
	var loc domain.CourierLocation
	err := r.collection.FindOne(ctx, filter).Decode(&loc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &loc, nil
}