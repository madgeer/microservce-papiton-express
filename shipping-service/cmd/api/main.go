package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
	handlerHttp "github.com/madgeer/papiton-express/shipping-service/internal/handler/http"
	mongoRepo "github.com/madgeer/papiton-express/shipping-service/internal/repository/mongo"
	"github.com/madgeer/papiton-express/shipping-service/internal/repository/postgres"
	"github.com/madgeer/papiton-express/shipping-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func initPostgres() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5433") // Port local luar docker
	user := getEnv("DB_USER", "user")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "shipping_test_db")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initMongo(ctx context.Context) (*mongo.Client, error) {
	uri := getEnv("MONGO_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func main() {
	log.Println("Memulai Shipping & Dispatch Service...")
	ctx := context.Background()

	var pgDB *sql.DB
	var mongoClient *mongo.Client
	var err error

	// 1. Hubungkan ke PostgreSQL
	pgDB, err = initPostgres()
	if err != nil {
		log.Printf("Peringatan: Gagal terhubung ke PostgreSQL: %v. Menggunakan mock/dummy.\n", err)
	} else {
		log.Println("Berhasil terhubung ke PostgreSQL Database!")
		defer pgDB.Close()
	}

	// 2. Hubungkan ke MongoDB
	mongoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	mongoClient, err = initMongo(mongoCtx)
	if err != nil {
		log.Printf("Peringatan: Gagal terhubung ke MongoDB: %v. Log lokasi real-time di-skip.\n", err)
	} else {
		log.Println("Berhasil terhubung ke MongoDB!")
		defer mongoClient.Disconnect(ctx)
	}

	// 3. Inisialisasi Repository
	var courierRepo domain.CourierRepository
	var dispatchRepo domain.DispatchRepository
	var locationRepo domain.LocationRepository

	if pgDB != nil {
		courierRepo = postgres.NewCourierRepository(pgDB)
		dispatchRepo = postgres.NewDispatchRepository(pgDB)
	}
	if mongoClient != nil {
		locationRepo = mongoRepo.NewLocationRepository(mongoClient.Database("shipping_db"))
	}

	// 4. Inisialisasi Service & Handler
	dispatchSvc := service.NewDispatchService(courierRepo, locationRepo, dispatchRepo)
	dispatchHandler := handlerHttp.NewDispatchHandler(dispatchSvc)

	// 5. Daftarkan rute HTTP
	http.HandleFunc("/dispatch", dispatchHandler.AutoDispatch)

	// 6. Jalankan Server HTTP
	port := getEnv("PORT", "8080")
	log.Printf("Server berjalan di http://localhost:%s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}