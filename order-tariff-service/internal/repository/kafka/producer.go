package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"order-tariff-service/internal/domain"

	"github.com/segmentio/kafka-go"
)

type orderEventPublisher struct {
	writer *kafka.Writer
}

// NewOrderEventPublisher membuat instance baru OrderEventPublisher yang tangguh dan tidak memblokir (asynchronous).
func NewOrderEventPublisher(topic string) domain.OrderEventPublisher {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	log.Printf("[Kafka Producer] Menginisialisasi Kafka writer pada broker: %s, topik: %s", broker, topic)

	// Inisialisasi writer dengan konfigurasi ketahanan (resiliency) tinggi
	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 1 * time.Second,  // Timeout pendek agar tidak hang saat broker mati
		RequiredAcks: kafka.RequireNone, // Mengutamakan performa dan ketahanan fallback
		Async:        true,             // Asynchronous agar tidak memblokir request utama user
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("[Kafka Producer Warning] Kesalahan asinkron Kafka: "+msg, args...)
		}),
	}

	return &orderEventPublisher{
		writer: writer,
	}
}

// PublishOrderCreated mengirimkan event order.created ke topik Kafka dengan payload yang kompatibel dengan Notification Service.
func (p *orderEventPublisher) PublishOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	if p == nil || p.writer == nil {
		log.Println("[Kafka Producer] WARNING: Kafka writer tidak aktif, mengabaikan publish")
		return nil
	}

	// Buat payload IncomingEvent yang sesuai dengan skema Notification Service
	payload := map[string]interface{}{
		"event_id":    "EVT-ORD-" + event.AWB,
		"event_type":  "order.created",
		"user_id":     event.Email, // Menggunakan Email sebagai UserID untuk pengiriman notifikasi
		"awb":         event.AWB,
		"occurred_at": event.Timestamp,
		"metadata": map[string]string{
			"status": event.Status,
		},
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Kafka Producer] Gagal melakukan serialisasi event: %v", err)
		return err
	}

	// Gunakan context dengan timeout pendek
	pubCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	log.Printf("[Kafka Producer] Mencoba mempublikasikan event order.created untuk AWB %s...", event.AWB)

	err = p.writer.WriteMessages(pubCtx, kafka.Message{
		Key:   []byte(event.AWB),
		Value: bytes,
	})

	if err != nil {
		// Log warning saja (tidak merusak transaksi database / return error) untuk menjaga resiliency fallback
		log.Printf("[Kafka Producer] WARNING: Gagal mengirim event ke Kafka broker (broker kemungkinan offline): %v", err)
		return nil
	}

	log.Printf("[Kafka Producer] Berhasil memasukkan event order.created untuk AWB %s ke dalam buffer asinkron.", event.AWB)
	return nil
}
