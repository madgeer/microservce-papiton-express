//go:build functional

package mongo_test

import (
	"context"
	"testing"
	"time"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
	repoMongo "github.com/madgeer/papiton-express/shipping-service/internal/repository/mongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestLocationRepository_UpdateLocation_Functional(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Gagal koneksi ke Mongo: %v", err)
	}
	db := client.Database("shipping_test_db")

	repo := repoMongo.NewLocationRepository(db)

	loc := &domain.CourierLocation{
		CourierID: "CUR-TEST",
		Latitude:  -6.917464,
		Longitude: 107.619123,
		Timestamp: time.Now(),
	}

	err = repo.UpdateLocation(ctx, loc)

	// Assert: Kita mengharapkan insert berhasil, tapi fungsi mengembalikan ErrNotImplemented
	if err != nil {
		t.Fatalf("Ekspektasi insert berhasil, tapi mendapatkan error: %v", err)
	}
}