package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/lib/pq"
)

// DB menyimpan instance koneksi database secara global untuk testing
var DB *sql.DB
var pgContainer *postgres.PostgresContainer

/* SetupTestDB menginisialisasi koneksi ke database menggunakan Testcontainers */
func SetupTestDB() *sql.DB {
	ctx := context.Background()

	// Mulai container PostgreSQL baru khusus untuk testing secara terisolasi
	container, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("papiton_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(15*time.Second),
		),
	)

	if err != nil {
		log.Printf("Gagal menjalankan container test DB: %v", err)
		return nil
	}

	pgContainer = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("Gagal mendapatkan string koneksi: %v", err)
		return nil
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Gagal membuka koneksi database: %v", err)
		return nil
	}

	// Memastikan database benar-benar bisa dihubungi
	err = db.Ping()
	if err != nil {
		log.Printf("Database container tidak merespons: %v", err)
		return nil
	}

	// Jalankan migrasi schema untuk membuat struktur tabel
	var migrationSQL []byte
	var readErr error
	pathsToTry := []string{
		"../../migrations/001_init_schema.up.sql",
		"../migrations/001_init_schema.up.sql",
		"migrations/001_init_schema.up.sql",
	}
	for _, p := range pathsToTry {
		migrationSQL, readErr = os.ReadFile(p)
		if readErr == nil {
			break
		}
	}
	if readErr != nil {
		log.Printf("Gagal membaca file migrasi: %v", readErr)
		return nil
	}

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Printf("Gagal mengeksekusi file migrasi: %v", err)
		return nil
	}

	fmt.Println("Berhasil terhubung ke Test Database PostgreSQL via Testcontainers dan menjalankan migrasi!")
	DB = db
	return db
}

/* CleanTestDB digunakan untuk membersihkan instance testcontainer setelah test selesai */
func CleanTestDB() {
	if DB != nil {
		DB.Close()
	}
	if pgContainer != nil {
		if err := pgContainer.Terminate(context.Background()); err != nil {
			log.Printf("Gagal mematikan container Test DB: %s", err)
		} else {
			fmt.Println("Container Test Database berhasil dimatikan.")
		}
	}
}
