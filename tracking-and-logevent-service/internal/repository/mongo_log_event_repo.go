package repository

import (
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
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
	// TODO: implementasi query insert mongodb
	return ErrDBNotImplemented
}
