package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/handler"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
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
	log.Println("Memulai Tracking & Log Event Service...")
	ctx := context.Background()

	// 1. Hubungkan ke MongoDB
	mongoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	mongoClient, err := initMongo(mongoCtx)
	var db *mongo.Database
	if err != nil {
		log.Printf("Peringatan: Gagal terhubung ke MongoDB: %v. Menggunakan repositori fallback.\n", err)
	} else {
		log.Println("Berhasil terhubung ke MongoDB!")
		defer mongoClient.Disconnect(ctx)
		db = mongoClient.Database(getEnv("DB_NAME", "tracking_db"))
	}

	// 2. Setup Layer Clean Architecture (Repository -> Service -> Handler)
	logEventRepo := repository.NewMongoLogEventRepo(db)
	trackingRepo := repository.NewMongoTrackingRepo(db)

	logEventSvc := service.NewLogEventService(logEventRepo)
	_ = logEventSvc
	trackingSvc := service.NewTrackingService(trackingRepo)

	trackingHandler := handler.NewTrackingAPIHandler(trackingSvc)
	logEventAPIHandler := handler.NewLogEventAPIHandler(logEventSvc)

	// 3. Mendaftarkan HTTP Route Handler
	http.HandleFunc("/api/v1/tracking", trackingHandler.GetHistory)
	http.HandleFunc("/api/v1/tracking/scan", logEventAPIHandler.ScanLog)
	http.HandleFunc("/api/v1/tracking/logs", logEventAPIHandler.GetLogs)

	// 4. Membaca port dinamis dari Environment Variable (menghindari port conflict)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Port bawaan di dalam Docker container
	}

	log.Printf("Server berjalan di http://localhost:%s\n", port)

	// 5. Menyalakan Server HTTP
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Gagal menjalankan server HTTP: %v", err)
	}
}