package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"order-tariff-service/internal/handler"
	"order-tariff-service/internal/repository/postgres"
	"order-tariff-service/internal/service"
)

func main() {
	fmt.Println("Memulai Order & Tariff Service...")

	// 1. Setup Layer Database (Koneksi ke PostgreSQL)
	db, err := postgres.InitDB()
	if err != nil {
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	log.Println("Database PostgreSQL terkoneksi dengan sukses!")

	// 2. Setup Layer Clean Architecture (Repository -> Service -> Handler)
	repo := postgres.NewOrderRepository(db)
	svc := service.NewOrderService(repo)
	orderHandler := handler.NewOrderHandler(svc)

	// 3. Mendaftarkan HTTP Route Handler
	http.HandleFunc("/api/v1/orders", orderHandler.HandleCreateOrder)

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
