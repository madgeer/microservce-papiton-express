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

	"papiton/notification-service/internal/consumer"
	"papiton/notification-service/internal/dispatcher"
	"papiton/notification-service/internal/handler"
	"papiton/notification-service/internal/processor"
	"papiton/notification-service/internal/provider"
	"papiton/notification-service/internal/repository"
)

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
			cid = fmt.Sprintf("ntf-%d", time.Now().UnixNano())
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
	// ── Konfigurasi dari environment variable ─────────────────────────────────
	apiKey := getEnv("API_KEY", "")
	rpm, _ := strconv.Atoi(getEnv("RATE_LIMIT_RPM", "0"))
	if rpm <= 0 {
		rpm = 100
	}
	rl := newRateLimiter(rpm)
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "notification-service-group")
	fcmProjectID := getEnv("FCM_PROJECT_ID", "")
	fcmServiceAccountJSON := getEnv("FCM_SERVICE_ACCOUNT_JSON", "")
	smtpHost := getEnv("SMTP_HOST", "smtp.papiton.id")
	fromEmail := getEnv("FROM_EMAIL", "noreply@papiton.id")
	smtpPortStr := getEnv("SMTP_PORT", "587")
	smtpPort := 587
	fmt.Sscanf(smtpPortStr, "%d", &smtpPort)
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASSWORD", "")

	topics := []string{
		"papiton.events.order",
		"papiton.events.shipping",
		"papiton.events.tracking",
	}

	// ── Koneksi Database PostgreSQL ──────────────────────────────────────────
	dbHost := getEnv("DB_HOST", "notification-db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "user")
	dbPass := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "notification_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("Peringatan: Gagal melakukan ping database: %v", err)
	}

	// ── Inisialisasi komponen ─────────────────────────────────────────────────
	proc := processor.NewMessageProcessor()
	emailProv := provider.NewEmailProvider(smtpHost, smtpPort, fromEmail, smtpUser, smtpPass)
	pushProv := provider.NewPushProvider(fcmProjectID, fcmServiceAccountJSON)
	repo := repository.NewPostgresNotificationRepository(db)
	disp := dispatcher.NewDispatcher(emailProv, pushProv, repo)

	kafkaConsumer := consumer.NewKafkaConsumer(
		kafkaBroker,
		kafkaGroupID,
		topics,
		proc,
		disp,
		repo,
	)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Menerima sinyal shutdown, menghentikan service...")
		cancel()
	}()

	// Jalankan Kafka Consumer di latar belakang
	go func() {
		log.Println("PAPITON Express - Notification Service (Kafka Consumer) dimulai")
		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()

	// Inisialisasi API Handler & Server HTTP
	notificationHandler := handler.NewNotificationAPIHandler(disp, repo)
	chain := func(h http.HandlerFunc) http.HandlerFunc {
		return withCorrelationID(requireAPIKey(apiKey, withRateLimit(rl, h)))
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"notification-messaging-service"}`))
	})
	http.HandleFunc("/api/v1/notifications/logs", chain(notificationHandler.GetLogs))
	http.HandleFunc("/api/v1/notifications/send-direct", chain(notificationHandler.SendDirect))

	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP Server berjalan di http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v\n", err)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown HTTP Server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)

	log.Println("Menutup Kafka Consumer...")
	if err := kafkaConsumer.Close(); err != nil {
		log.Printf("Gagal menutup kafkaConsumer: %v", err)
	}

	log.Println("Layanan dihentikan sepenuhnya.")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
