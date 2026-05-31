package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"warehouse-inventory-service/internal/handler"
	"warehouse-inventory-service/internal/repository"
	"warehouse-inventory-service/internal/service"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func initDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return nil, fmt.Errorf("DB_HOST environment variable is not set")
	}
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "papiton_warehouse")

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

func main() {
	fmt.Println("Memulai Warehouse & Inventory Service...")

	var inboundRepo repository.InboundRepository
	var manifestRepo repository.ManifestRepository

	// Menghubungkan ke PostgreSQL jika DB_HOST terkonfigurasi, jika tidak gunakan Dummy
	db, err := initDB()
	if err != nil {
		log.Printf("Peringatan inisialisasi database: %v. Berjalan dalam mode Dummy.\n", err)
		inboundRepo = repository.NewDummyInboundRepo()
		manifestRepo = repository.NewPostgresManifestRepo(nil) // fallback
	} else {
		log.Println("Berhasil terhubung ke PostgreSQL Database!")
		defer db.Close()
		inboundRepo = repository.NewPostgresInboundRepo(db)
		manifestRepo = repository.NewPostgresManifestRepo(db)
	}

	// 2. Setup Layer Service (Inject Repository ke dalam Service)
	inboundSvc := service.NewInboundService(inboundRepo)
	manifestSvc := service.NewManifestService(manifestRepo)

	// 3. Setup Layer Handler (Inject Service ke dalam Handler)
	inboundHandler := handler.NewInboundHandler(inboundSvc)
	manifestHandler := handler.NewManifestHandler(manifestSvc)

	// 4. Mendaftarkan Rute HTTP (Routing/Endpoint)
	http.HandleFunc("/api/v1/inbound", inboundHandler.HandleProcessInbound)
	http.HandleFunc("/api/v1/manifest/create", manifestHandler.HandleCreateManifest)
	http.HandleFunc("/api/v1/manifest/add", manifestHandler.HandleAddToManifest)
	http.HandleFunc("/api/v1/manifest/update", manifestHandler.HandleUpdateManifest)

	// 5. Menjalankan Server HTTP dengan Port dinamis
	port := getEnv("PORT", "8080")
	fmt.Printf("Server berjalan di http://localhost:%s\n", port)
	
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
