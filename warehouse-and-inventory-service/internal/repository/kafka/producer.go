package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"warehouse-inventory-service/internal/service"

	"github.com/segmentio/kafka-go"
)

type warehouseEventPublisher struct {
	writer *kafka.Writer
}

// NewWarehouseEventPublisher membuat instance baru WarehouseEventPublisher yang tangguh dan asinkron.
func NewWarehouseEventPublisher(topic string) service.WarehouseEventPublisher {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	log.Printf("[Warehouse Kafka Producer] Menginisialisasi Kafka writer pada broker: %s, topik: %s", broker, topic)

	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 1 * time.Second,
		RequiredAcks: kafka.RequireNone,
		Async:        true,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("[Warehouse Kafka Producer Warning] Kesalahan asinkron Kafka: "+msg, args...)
		}),
	}

	return &warehouseEventPublisher{
		writer: writer,
	}
}

// PublishInboundProcessed mengirimkan event inventory.inbound / package.in_transit ke topik papiton.events.tracking.
func (p *warehouseEventPublisher) PublishInboundProcessed(ctx context.Context, resi string, warehouseID string) error {
	if p == nil || p.writer == nil {
		log.Println("[Warehouse Kafka Producer] WARNING: Kafka writer tidak aktif, mengabaikan publish")
		return nil
	}

	// Payload kompatibel dengan Notification Service (memetakan inbound ke event package.in_transit)
	payload := map[string]interface{}{
		"event_id":    "EVT-INB-" + resi + "-" + time.Now().Format("20060102150405"),
		"event_type":  "package.in_transit", // Menggunakan tipe event yang dipahami oleh Notification Service
		"user_id":     "customer@papiton.id",  // Menggunakan email default sebagai UserID fallback
		"awb":         resi,
		"occurred_at": time.Now(),
		"metadata": map[string]string{
			"location": "Warehouse " + warehouseID,
			"status":   "inventory.inbound",
		},
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Warehouse Kafka Producer] Gagal melakukan serialisasi event: %v", err)
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	log.Printf("[Warehouse Kafka Producer] Mencoba mempublikasikan event tracking untuk Resi %s...", resi)

	err = p.writer.WriteMessages(pubCtx, kafka.Message{
		Key:   []byte(resi),
		Value: bytes,
	})

	if err != nil {
		log.Printf("[Warehouse Kafka Producer] WARNING: Gagal mengirim event ke Kafka broker (broker offline): %v", err)
		return nil
	}

	log.Printf("[Warehouse Kafka Producer] Berhasil memasukkan event tracking untuk Resi %s ke buffer asinkron.", resi)
	return nil
}
