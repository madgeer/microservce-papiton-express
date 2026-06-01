package repository

import (
	"context"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
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
	if r.db == nil {
		return nil, ErrDBNotImplemented
	}
	collection := r.db.Collection("tracking_logs")
	filter := bson.M{"resi_id": resiID}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var logs []model.TrackingLog
	if err := cursor.All(context.Background(), &logs); err != nil {
		return nil, err
	}

	history := &model.TrackingHistory{
		ResiID:  resiID,
		History: logs,
	}
	return history, nil
}
