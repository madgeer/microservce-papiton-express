package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"papiton/notification-service/internal/consumer"
	"papiton/notification-service/internal/dispatcher"
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

	// ── Inisialisasi komponen ─────────────────────────────────────────────────
	proc := processor.NewMessageProcessor()
	emailProv := provider.NewEmailProvider(smtpHost, 587, fromEmail)
	pushProv := provider.NewPushProvider(fcmKey)
	repo := repository.NewPostgresNotificationRepository()
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

	log.Println("PAPITON Express - Notification Service dimulai")
	if err := kafkaConsumer.Start(ctx); err != nil {
		log.Fatalf("Kafka consumer error: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
