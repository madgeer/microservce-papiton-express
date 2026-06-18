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

// IdempotencyChecker mendefinisikan interface untuk mengecek keunikan event_id
type IdempotencyChecker interface {
	IsEventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
}

// KafkaConsumer mengelola konsumsi event dari Kafka
type KafkaConsumer struct {
	brokers    string
	groupID    string
	topics     []string
	processor  EventProcessor
	dispatcher MessageDispatcher
	checker    IdempotencyChecker
	dlqWriter  *kafka.Writer
}

func NewKafkaConsumer(
	brokers string,
	groupID string,
	topics []string,
	processor EventProcessor,
	dispatcher MessageDispatcher,
	checker IdempotencyChecker,
) *KafkaConsumer {
	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
		Topic:        "papiton.dlq.events",
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 2 * time.Second,
	}

	return &KafkaConsumer{
		brokers:    brokers,
		groupID:    groupID,
		topics:     topics,
		processor:  processor,
		dispatcher: dispatcher,
		checker:    checker,
		dlqWriter:  dlqWriter,
	}
}

func (kc *KafkaConsumer) Close() error {
	if kc.dlqWriter != nil {
		return kc.dlqWriter.Close()
	}
	return nil
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

	// Cek Idempotensi Event
	if event.EventID != "" && kc.checker != nil {
		processed, err := kc.checker.IsEventProcessed(ctx, event.EventID)
		if err != nil {
			log.Printf("[KafkaConsumer Error] Gagal memeriksa idempotensi event %s: %v. Lanjut memproses...", event.EventID, err)
		} else if processed {
			log.Printf("[KafkaConsumer Idempotensi] Event %s sudah pernah diproses sebelumnya. Silently discarded.", event.EventID)
			return
		}
	}

	notification, err := kc.processor.Process(event)
	if err != nil {
		log.Printf("[KafkaConsumer] Gagal memproses event %s: %v", event.EventID, err)
		return
	}

	// 1. Eksekusi pertama kali
	err = kc.dispatcher.Dispatch(ctx, *notification)
	if err == nil {
		// Tandai event telah sukses diproses
		if event.EventID != "" && kc.checker != nil {
			_ = kc.checker.MarkEventProcessed(ctx, event.EventID)
		}
		return
	}

	// 2. Retry Strategy: Exponential Backoff (Maksimal 5x percobaan ulang)
	log.Printf("[KafkaConsumer Warning] Gagal dispatch notifikasi awal untuk AWB %s: %v. Memulai strategi retry...", event.AWB, err)
	
	maxRetries := 5
	backoff := 1 * time.Second

	for i := 1; i <= maxRetries; i++ {
		log.Printf("[KafkaConsumer] Percobaan ulang ke-%d untuk event %s dalam %v...", i, event.EventID, backoff)
		time.Sleep(backoff)
		
		err = kc.dispatcher.Dispatch(ctx, *notification)
		if err == nil {
			log.Printf("[KafkaConsumer] Notifikasi berhasil terkirim pada percobaan ulang ke-%d.", i)
			if event.EventID != "" && kc.checker != nil {
				_ = kc.checker.MarkEventProcessed(ctx, event.EventID)
			}
			return
		}
		
		backoff *= 2 // Exponential backoff: 1s, 2s, 4s, 8s, 16s
	}

	// 3. Dead Letter Queue (DLQ): Jika 5x percobaan tetap gagal, lempar pesan ke topik papiton.dlq.events
	log.Printf("[KafkaConsumer Error] Seluruh %d percobaan ulang gagal untuk event %s. Mengirim event ke Dead Letter Queue (DLQ)...", maxRetries, event.EventID)

	dlqPayload := map[string]interface{}{
		"original_event": event,
		"error_reason":   err.Error(),
		"failed_at":      time.Now().Format(time.RFC3339),
	}

	dlqBytes, errMarshal := json.Marshal(dlqPayload)
	if errMarshal != nil {
		log.Printf("[KafkaConsumer Error] Gagal marshal payload DLQ: %v", errMarshal)
		return
	}

	pubCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	errDLQ := kc.dlqWriter.WriteMessages(pubCtx, kafka.Message{
		Key:   []byte(event.AWB),
		Value: dlqBytes,
	})

	if errDLQ != nil {
		log.Printf("[KafkaConsumer DLQ Error] Gagal mengirim pesan ke topik papiton.dlq.events: %v", errDLQ)
	} else {
		log.Printf("[KafkaConsumer DLQ Success] Berhasil memindahkan pesan event %s ke Dead Letter Queue (Topik: papiton.dlq.events)", event.EventID)
		if event.EventID != "" && kc.checker != nil {
			_ = kc.checker.MarkEventProcessed(ctx, event.EventID)
		}
	}
}
