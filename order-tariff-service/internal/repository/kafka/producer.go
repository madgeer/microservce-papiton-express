package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
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
	// Mendukung comma-separated list untuk Kafka cluster multi-broker
	brokerList := strings.Split(broker, ",")
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 1 * time.Second,
		RequiredAcks: kafka.RequireOne, // Minimal 1 broker mengkonfirmasi — lebih aman dengan cluster
		Async:        true,
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

	// Payload diperkaya agar consumer (ETL, Notification) tidak perlu query balik ke order-db.
	payload := map[string]interface{}{
		"event_id":    "EVT-ORD-" + event.AWB,
		"event_type":  "order.created",
		"user_id":     event.Email,
		"awb":         event.AWB,
		"occurred_at": event.Timestamp,
		"metadata": map[string]interface{}{
			"status":         event.Status,
			"service_type":   event.ServiceType,
			"has_insurance":  event.HasInsurance,
			"has_packing":    event.HasPacking,
			"sender_city":    event.SenderCity,
			"recipient_city": event.RecipientCity,
			"package_weight": event.PackageWeight,
			"package_length": event.PackageLength,
			"package_width":  event.PackageWidth,
			"package_height": event.PackageHeight,
			"tarif_total":    event.TarifTotal,
			"distance_km":    event.DistanceKM,
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
