package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type dispatchEventPublisher struct {
	writer *kafka.Writer
}

// NewDispatchEventPublisher membuat instance baru DispatchEventPublisher yang resilient dan asinkron.
func NewDispatchEventPublisher(topic string) domain.DispatchEventPublisher {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	log.Printf("[Shipping Kafka Producer] Menginisialisasi Kafka writer pada broker: %s, topik: %s", broker, topic)

	brokerList := strings.Split(broker, ",")
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokerList...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           1 * time.Second,
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,
		AllowAutoTopicCreation: true,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			log.Printf("[Shipping Kafka Producer Warning] Kesalahan asinkron Kafka: "+msg, args...)
		}),
	}

	return &dispatchEventPublisher{
		writer: writer,
	}
}

// PublishDispatchAssigned mempublikasikan event penugasan kurir ke topik papiton.events.shipping.
func (p *dispatchEventPublisher) PublishDispatchAssigned(ctx context.Context, dispatch *domain.Dispatch) error {
	if p == nil || p.writer == nil {
		log.Println("[Shipping Kafka Producer] WARNING: Kafka writer tidak aktif, mengabaikan publish")
		return nil
	}

	// Payload diperkaya dengan vehicle_type agar ETL tidak perlu query shipping-db.
	payload := map[string]interface{}{
		"event_id":    "EVT-DSP-" + dispatch.ID + "-" + time.Now().Format("20060102150405"),
		"event_type":  "package.picked_up",
		"user_id":     "customer@papiton.id",
		"awb":         dispatch.OrderID,
		"occurred_at": time.Now(),
		"metadata": map[string]string{
			"courier_id":        dispatch.CourierID,
			"vehicle_type":      dispatch.VehicleType,
			"status":            "dispatch.assigned",
			"route_instruction": dispatch.RouteInstruction,
		},
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Shipping Kafka Producer] Gagal melakukan serialisasi event: %v", err)
		return err
	}

	pubCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	log.Printf("[Shipping Kafka Producer] Mencoba mempublikasikan event dispatch untuk AWB %s...", dispatch.OrderID)

	err = p.writer.WriteMessages(pubCtx, kafka.Message{
		Key:   []byte(dispatch.OrderID),
		Value: bytes,
	})

	if err != nil {
		log.Printf("[Shipping Kafka Producer] WARNING: Gagal mengirim event ke Kafka broker (broker offline): %v", err)
		return nil
	}

	log.Printf("[Shipping Kafka Producer] Berhasil memasukkan event dispatch untuk AWB %s ke buffer asinkron.", dispatch.OrderID)
	return nil
}
