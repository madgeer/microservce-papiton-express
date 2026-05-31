package repository

import (
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

/* MongoTrackingRepo adalah implementasi TrackingRepository menggunakan MongoDB */
type MongoTrackingRepo struct {
	db *mongo.Database
}

/* NewMongoTrackingRepo membuat instance baru untuk MongoTrackingRepo */
func NewMongoTrackingRepo(db *mongo.Database) *MongoTrackingRepo {
	return &MongoTrackingRepo{db: db}
}

func (r *MongoTrackingRepo) GetResiHistory(resiID string) (*model.TrackingHistory, error) {
	// TODO: implementasi query find mongodb
	return nil, ErrDBNotImplemented
}
