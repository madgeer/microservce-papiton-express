package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

func main() {
	// ── Konfigurasi dari environment variable ─────────────────────────────────
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "notification-service-group")
	fcmKey := getEnv("FCM_SERVER_KEY", "")
	smtpHost := getEnv("SMTP_HOST", "smtp.papiton.id")
	fromEmail := getEnv("FROM_EMAIL", "noreply@papiton.id")

	topics := []string{
		"papiton.events.order",
		"papiton.events.shipping",
		"papiton.events.tracking",
	}

	// ── Koneksi Database PostgreSQL ──────────────────────────────────────────
	dbHost := getEnv("DB_HOST", "shipping-db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "user")
	dbPass := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "shipping_test_db")

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
	emailProv := provider.NewEmailProvider(smtpHost, 587, fromEmail)
	pushProv := provider.NewPushProvider(fcmKey)
	repo := repository.NewPostgresNotificationRepository(db)
	disp := dispatcher.NewDispatcher(emailProv, pushProv, repo)

	kafkaConsumer := consumer.NewKafkaConsumer(
		kafkaBroker,
		kafkaGroupID,
		topics,
		proc,
		disp,
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
	http.HandleFunc("/api/v1/notifications/logs", notificationHandler.GetLogs)
	http.HandleFunc("/api/v1/notifications/send-direct", notificationHandler.SendDirect)

	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr: ":" + port,
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
	log.Println("Layanan dihentikan sepenuhnya.")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
