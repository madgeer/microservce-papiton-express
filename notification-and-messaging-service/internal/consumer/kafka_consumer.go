package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"papiton/notification-service/internal/model"

	"github.com/segmentio/kafka-go"
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
type KafkaConsumer struct {
	brokers    string
	groupID    string
	topics     []string
	processor  EventProcessor
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

// Start memulai loop konsumsi event dari Kafka secara riil menggunakan goroutine untuk setiap topik.
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	log.Printf("[KafkaConsumer] Mulai listening pada topics: %v", kc.topics)

	for _, topic := range kc.topics {
		go func(t string) {
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:        []string{kc.brokers},
				GroupID:        kc.groupID,
				Topic:          t,
				MinBytes:       10e3, // 10KB
				MaxBytes:       10e6, // 10MB
				CommitInterval: 1 * time.Second,
			})
			defer r.Close()

			log.Printf("[KafkaConsumer] Goroutine aktif untuk mendengarkan topik: %s", t)

			for {
				select {
				case <-ctx.Done():
					log.Printf("[KafkaConsumer] Menghentikan pendengar untuk topik: %s", t)
					return
				default:
					m, err := r.ReadMessage(ctx)
					if err != nil {
						// Jika context ditutup, langsung keluar secara anggun
						if ctx.Err() != nil {
							return
						}
						log.Printf("[KafkaConsumer] Gagal membaca pesan dari topik %s: %v", t, err)
						time.Sleep(1 * time.Second)
						continue
					}

					log.Printf("[KafkaConsumer] Menerima event baru dari topik %s", t)
					kc.handleMessage(ctx, m.Value)
				}
			}
		}(topic)
	}

	<-ctx.Done()
	log.Println("[KafkaConsumer] Seluruh goroutine pendengar dihentikan.")
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
