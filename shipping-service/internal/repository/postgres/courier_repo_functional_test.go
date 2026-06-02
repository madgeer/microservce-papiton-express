//go:build functional

// Wajib ada tag ini agar tidak ikut tereksekusi di tahap Unit Test (Tahap 2 Pipeline)

package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/madgeer/papiton-express/shipping-service/internal/repository/postgres"
)

func TestCourierRepository_GetAvailableByZone_Functional(t *testing.T) {
	// Dalam Red Phase murni, kita bisa asumsikan koneksi database sudah ada
	// Jika gagal connect, test juga otomatis fail (yang mana valid di fase Red)
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5433/shipping_test_db?sslmode=disable")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	repo := postgres.NewCourierRepository(db)
	ctx := context.Background()

	couriers, err := repo.GetAvailableByZone(ctx, "Bandung")

	// Assert: Kita mengharapkan data, tapi kode produksi mengembalikan ErrNotImplemented
	if err != nil {
		t.Fatalf("Ekspektasi tidak ada error, tapi gagal dengan: %v", err)
	}
	if len(couriers) == 0 {
		t.Fatalf("Ekspektasi mendapatkan data armada, namun kosong")
	}
}