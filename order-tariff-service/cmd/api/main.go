package main

import (
	"context"
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

	"order-tariff-service/internal/handler"
	"order-tariff-service/internal/repository/kafka"
	"order-tariff-service/internal/repository/postgres"
	"order-tariff-service/internal/repository/redis"
	"order-tariff-service/internal/service"
)

// rateLimiter implementasi sliding window per-IP tanpa dependency eksternal.
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(rpm int) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    rpm,
		window:   time.Minute,
	}
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
			cid = fmt.Sprintf("ord-%d", time.Now().UnixNano())
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
	fmt.Println("Memulai Order & Tariff Service...")

	apiKey := os.Getenv("API_KEY")
	rpm, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	if rpm <= 0 {
		rpm = 100
	}
	rl := newRateLimiter(rpm)

	// 1. Setup Layer Database (Koneksi ke PostgreSQL)
	db, err := postgres.InitDB()
	if err != nil {
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()
	log.Println("Database PostgreSQL terkoneksi dengan sukses!")

	// 1.1. Setup Layer Cache (Koneksi ke Redis)
	rdb, err := redis.InitRedis()
	if err != nil {
		log.Printf("[Warning] Gagal inisialisasi Redis: %v. Berjalan dengan fallback database.", err)
	} else {
		defer rdb.Close()
	}

	// 2. Setup Layer Clean Architecture (Repository -> Service -> Handler)
	repo := postgres.NewOrderRepository(db, rdb)
	kafkaPub := kafka.NewOrderEventPublisher("papiton.events.order")
	svc := service.NewOrderService(repo, kafkaPub)
	orderHandler := handler.NewOrderHandler(svc)

	// 3. Mendaftarkan HTTP Route Handler
	chain := func(h http.HandlerFunc) http.HandlerFunc {
		return withCorrelationID(requireAPIKey(apiKey, withRateLimit(rl, h)))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"order-tariff-service"}`))
	})
	http.HandleFunc("/api/v1/orders", chain(orderHandler.HandleCreateOrder))
	http.HandleFunc("/api/v1/orders/get", chain(orderHandler.HandleGetOrders))
	http.HandleFunc("/api/v1/tariff/calculate", chain(orderHandler.HandleCalculateTariff))

	// 4. Membaca port dinamis dari Environment Variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 5. HTTP Server dengan timeout
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server berjalan di http://localhost:%s (rate limit: %d req/menit)\n", port, rpm)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gagal menjalankan server HTTP: %v", err)
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
