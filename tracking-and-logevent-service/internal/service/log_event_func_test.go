//go:build functional

package service

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
)

func TestProcessLog_DB_Failed(t *testing.T) {
	// Persiapan Koneksi MongoDB (Simulasi DB Test)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err == nil {
		defer client.Disconnect(ctx)
	} else {
		t.Fatal("Database MongoDB test tidak tersedia, functional test gagal setup")
	}

	db := client.Database("tracking_db")

	// Persiapan Service & Repository E2E
	repo := repository.NewMongoLogEventRepo(db)
	svc := NewLogEventService(repo, nil)

	// Eksekusi fungsi end-to-end
	dummyLog := model.TrackingLog{ResiID: "RESI-123456", ActivityCode: "DELIVERED"}
	err = svc.ProcessLog(dummyLog)
	assert.NoError(t, err)
}
