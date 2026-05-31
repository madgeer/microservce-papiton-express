package main

import (
	"fmt"
	"log"
	"order-tariff-service/internal/repository/postgres"
)

func main() {
	fmt.Println("Order & Tariff Service sedang berjalan....")

	//mencoba hubungkan ke PostgreSQL
	db, err := postgres.InitDB()
	if err != nil {
		// jika gagal connect
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}

	defer db.Close()

	log.Println("Database terkoneksi!")
}
