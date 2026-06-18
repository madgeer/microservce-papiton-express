package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/segmentio/kafka-go"
)

type TrackingEventPublisher interface {
	PublishTrackingLog(ctx context.Context, trackingLog model.TrackingLog) error
}

type trackingEventPublisher struct {
	writer *kafka.Writer
}

func NewTrackingEventPublisher(topic string) TrackingEventPublisher {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	log.Printf("[Kafka Producer] Menginisialisasi Kafka writer pada broker: %s, topik: %s", broker, topic)

	brokerList := strings.Split(broker, ",")
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 1 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("[Kafka Producer Warning] Kesalahan asinkron Kafka: "+msg, args...)
		}),
	}

	return &trackingEventPublisher{
		writer: writer,
	}
}

func (p *trackingEventPublisher) PublishTrackingLog(ctx context.Context, trackingLog model.TrackingLog) error {
	if p == nil || p.writer == nil {
		log.Println("[Kafka Producer] WARNING: Kafka writer tidak aktif, mengabaikan publish")
		return nil
	}

	// Payload event
	payload := map[string]interface{}{
		"event_id":      "EVT-TRK-" + trackingLog.ResiID + "-" + time.Now().Format("20060102150405"),
		"event_type":    "package." + stringsToLower(trackingLog.ActivityCode),
		"resi_id":       trackingLog.ResiID,
		"activity_code": trackingLog.ActivityCode,
		"location_code": trackingLog.LocationCode,
		"timestamp":     trackingLog.Timestamp,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Kafka Producer] Gagal melakukan serialisasi event: %v", err)
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	log.Printf("[Kafka Producer] Mencoba mempublikasikan event tracking untuk AWB %s...", trackingLog.ResiID)

	err = p.writer.WriteMessages(pubCtx, kafka.Message{
		Key:   []byte(trackingLog.ResiID),
		Value: bytes,
	})

	if err != nil {
		log.Printf("[Kafka Producer] WARNING: Gagal mengirim event ke Kafka broker: %v", err)
		return nil
	}

	log.Printf("[Kafka Producer] Berhasil mengirim event tracking untuk AWB %s ke buffer asinkron.", trackingLog.ResiID)
	return nil
}

// Simple helper to avoid external strings import if not needed
func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
