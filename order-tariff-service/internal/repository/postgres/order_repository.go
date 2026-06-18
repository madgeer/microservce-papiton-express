package postgres

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// untuk koneksi ke database
func InitDB() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5434")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "admin123")
	dbname := getEnv("DB_NAME", "papiton_order_tariff_service_db")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi: %v", err)
	}

	//tes koneksi
	err = DB.Ping()
	if err != nil {
		return nil, fmt.Errorf("Ping gagal, Database tidak merespon:%v", err)
	}

	fmt.Println("Sukses! Terhubung ke PostgreSQL")

	// Aktifkan ekstensi PostGIS jika belum ada
	_, err = DB.Exec("CREATE EXTENSION IF NOT EXISTS postgis;")
	if err != nil {
		fmt.Printf("Peringatan: Gagal mengaktifkan ekstensi PostGIS: %v\n", err)
	} else {
		fmt.Println("Ekstensi PostGIS berhasil diaktifkan / sudah siap!")
	}

	return DB, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
