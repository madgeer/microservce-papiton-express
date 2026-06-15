package repository

import (
	"context"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/* MongoLogEventRepo adalah implementasi LogEventRepository menggunakan MongoDB */
type MongoLogEventRepo struct {
	db *mongo.Database
}

/* NewMongoLogEventRepo membuat instance baru untuk MongoLogEventRepo */
func NewMongoLogEventRepo(db *mongo.Database) *MongoLogEventRepo {
	return &MongoLogEventRepo{db: db}
}

func (r *MongoLogEventRepo) InsertLog(log model.TrackingLog) error {
	if r.db == nil {
		return ErrDBNotImplemented
	}
	collection := r.db.Collection("tracking_logs")
	_, err := collection.InsertOne(context.Background(), log)
	return err
}

func (r *MongoLogEventRepo) FindAllLogs() ([]model.TrackingLog, error) {
	if r.db == nil {
		return nil, ErrDBNotImplemented
	}
	collection := r.db.Collection("tracking_logs")
	
	opts := options.Find().SetSort(map[string]interface{}{"timestamp": -1}).SetLimit(100)
	cursor, err := collection.Find(context.Background(), map[string]interface{}{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var list []model.TrackingLog
	if err = cursor.All(context.Background(), &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []model.TrackingLog{}
	}
	return list, nil
}
