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

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/handler"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository/kafka"
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
			cid = fmt.Sprintf("trk-%d", time.Now().UnixNano())
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
	log.Println("Memulai Tracking & Log Event Service...")
	ctx := context.Background()

	apiKey := os.Getenv("API_KEY")
	rpm, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	if rpm <= 0 {
		rpm = 100
	}
	rl := newRateLimiter(rpm)

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

	kafkaPub := kafka.NewTrackingEventPublisher("papiton.events.tracking")
	logEventSvc := service.NewLogEventService(logEventRepo, kafkaPub)
	trackingSvc := service.NewTrackingService(trackingRepo)

	trackingHandler := handler.NewTrackingAPIHandler(trackingSvc)
	logEventAPIHandler := handler.NewLogEventAPIHandler(logEventSvc)

	// 3. Mendaftarkan HTTP Route Handler
	chain := func(h http.HandlerFunc) http.HandlerFunc {
		return withCorrelationID(requireAPIKey(apiKey, withRateLimit(rl, h)))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"tracking-logevent-service"}`))
	})
	http.HandleFunc("/api/v1/tracking", chain(trackingHandler.GetHistory))
	http.HandleFunc("/api/v1/tracking/scan", chain(logEventAPIHandler.ScanLog))
	http.HandleFunc("/api/v1/tracking/logs", chain(logEventAPIHandler.GetLogs))

	// 4. HTTP Server dengan timeout
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server berjalan di http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gagal menjalankan server HTTP: %v", err)
		}
	}()

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal shutdown, menghentikan server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Gagal graceful shutdown: %v", err)
	}
	log.Println("Server berhenti.")
}
