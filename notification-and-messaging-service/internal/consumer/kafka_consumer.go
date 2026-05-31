package consumer

import (
	"context"
	"encoding/json"
	"log"

	"papiton/notification-service/internal/model"
)

// EventProcessor adalah interface untuk memproses event yang masuk
type EventProcessor interface {
	Process(event model.IncomingEvent) (*model.NotificationMessage, error)
}

// MessageDispatcher adalah interface untuk mengirim notifikasi
type MessageDispatcher interface {
	Dispatch(ctx context.Context, msg model.NotificationMessage) error
}

// KafkaConsumer mengelola konsumsi event dari Kafka
// Catatan: Implementasi Kafka client nyata membutuhkan confluent-kafka-go
type KafkaConsumer struct {
	brokers   string
	groupID   string
	topics    []string
	processor EventProcessor
	dispatcher MessageDispatcher
}

func NewKafkaConsumer(
	brokers string,
	groupID string,
	topics []string,
	processor EventProcessor,
	dispatcher MessageDispatcher,
) *KafkaConsumer {
	return &KafkaConsumer{
		brokers:    brokers,
		groupID:    groupID,
		topics:     topics,
		processor:  processor,
		dispatcher: dispatcher,
	}
}

// Start memulai loop konsumsi event dari Kafka
// TODO: Ganti placeholder ini dengan implementasi confluent-kafka-go nyata
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	log.Printf("[KafkaConsumer] Mulai listening pada topics: %v", kc.topics)

	// Contoh loop konsumsi (pseudo-code):
	// for {
	//     select {
	//     case <-ctx.Done():
	//         return nil
	//     default:
	//         msg, err := consumer.ReadMessage(5 * time.Second)
	//         if err != nil { continue }
	//         kc.handleMessage(ctx, msg.Value)
	//     }
	// }

	return nil
}

// handleMessage memproses satu pesan Kafka
func (kc *KafkaConsumer) handleMessage(ctx context.Context, payload []byte) {
	var event model.IncomingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("[KafkaConsumer] Gagal parse payload: %v", err)
		return
	}

	notification, err := kc.processor.Process(event)
	if err != nil {
		log.Printf("[KafkaConsumer] Gagal memproses event %s: %v", event.EventID, err)
		return
	}

	if err := kc.dispatcher.Dispatch(ctx, *notification); err != nil {
		log.Printf("[KafkaConsumer] Gagal dispatch notifikasi untuk AWB %s: %v", event.AWB, err)
	}
}
