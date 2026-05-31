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
)

func TestGetHistory_DB_Failed(t *testing.T) {
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
	repo := repository.NewMongoTrackingRepo(db)
	svc := NewTrackingService(repo)

	// Eksekusi fungsi end-to-end
	_, err = svc.GetHistory("RESI-123456")
	_ = err // Abaikan error sementara karena dummy implementation

	// Verifikasi (Assertion)
	assert.Fail(t, "Functional test GetHistory gagal: Implementasi MongoDB E2E belum tersedia secara utuh")
}
