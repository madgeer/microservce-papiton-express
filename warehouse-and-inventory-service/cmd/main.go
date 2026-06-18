package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"warehouse-inventory-service/internal/handler"
	"warehouse-inventory-service/internal/repository"
	"warehouse-inventory-service/internal/repository/kafka"
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

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(rpm int) *rateLimiter {
	rl := &rateLimiter{requests: make(map[string][]time.Time), limit: rpm, window: time.Minute}
	go func() {
		for range time.Tick(time.Minute) {
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)
			for ip, times := range rl.requests {
				var recent []time.Time
				for _, t := range times {
					if t.After(cutoff) {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = recent
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.window)
	var recent []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rl.limit {
		return false
	}
	rl.requests[ip] = append(recent, time.Now())
	return true
}

func withRateLimit(rl *rateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(r) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Terlalu banyak request, coba lagi dalam 1 menit"})
			return
		}
		next(w, r)
	}
}

// withCorrelationID meneruskan atau membuat X-Correlation-ID pada setiap request/response.
func withCorrelationID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get("X-Correlation-ID")
		if cid == "" {
			cid = fmt.Sprintf("wh-%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Correlation-ID", cid)
		next(w, r)
	}
}

// requireAPIKey menolak request yang tidak menyertakan X-API-Key yang valid.
func requireAPIKey(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if apiKey != "" && r.Header.Get("X-API-Key") != apiKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized: X-API-Key tidak valid atau tidak ada"})
			return
		}
		next(w, r)
	}
}

func main() {
	fmt.Println("Memulai Warehouse & Inventory Service...")

	apiKey := os.Getenv("API_KEY")
	rpm, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	if rpm <= 0 {
		rpm = 100
	}
	rl := newRateLimiter(rpm)

	// Menghubungkan ke PostgreSQL — service ini tidak dapat berfungsi tanpa database
	db, err := initDB()
	if err != nil {
		log.Fatalf("FATAL: Gagal inisialisasi database: %v\n", err)
	}
	log.Println("Berhasil terhubung ke PostgreSQL Database!")
	defer db.Close()

	inboundRepo := repository.NewPostgresInboundRepo(db)
	manifestRepo := repository.NewPostgresManifestRepo(db)

	// 2. Setup Layer Service (Inject Repository ke dalam Service)
	kafkaPub := kafka.NewWarehouseEventPublisher("papiton.events.tracking")
	inboundSvc := service.NewInboundService(inboundRepo, kafkaPub)
	manifestSvc := service.NewManifestService(manifestRepo)

	// 3. Setup Layer Handler (Inject Service ke dalam Handler)
	inboundHandler := handler.NewInboundHandler(inboundSvc)
	manifestHandler := handler.NewManifestHandler(manifestSvc)

	// 4. Mendaftarkan Rute HTTP
	chain := func(h http.HandlerFunc) http.HandlerFunc {
		return withCorrelationID(requireAPIKey(apiKey, withRateLimit(rl, h)))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"warehouse-inventory-service"}`))
	})
	http.HandleFunc("/api/v1/inbound", chain(inboundHandler.HandleProcessInbound))
	http.HandleFunc("/api/v1/manifest/create", chain(manifestHandler.HandleCreateManifest))
	http.HandleFunc("/api/v1/manifest/add", chain(manifestHandler.HandleAddToManifest))
	http.HandleFunc("/api/v1/manifest/update", chain(manifestHandler.HandleUpdateManifest))

	// 5. HTTP Server dengan timeout
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("Server berjalan di http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gagal menjalankan server: %v", err)
		}
	}()

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal shutdown, menghentikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Gagal graceful shutdown: %v", err)
	}
	log.Println("Server berhenti.")
}
