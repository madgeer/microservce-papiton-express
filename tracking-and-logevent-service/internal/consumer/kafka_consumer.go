package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	brokers   string
	groupID   string
	topics    []string
	repo      repository.LogEventRepository
	trackRepo repository.TrackingRepository
	wg        sync.WaitGroup
}

func NewKafkaConsumer(
	brokers string,
	groupID string,
	topics []string,
	repo repository.LogEventRepository,
	trackRepo repository.TrackingRepository,
) *KafkaConsumer {
	return &KafkaConsumer{
		brokers:   brokers,
		groupID:   groupID,
		topics:    topics,
		repo:      repo,
		trackRepo: trackRepo,
	}
}

func (kc *KafkaConsumer) Start(ctx context.Context) error {
	log.Printf("[KafkaConsumer] Mulai listening pada topics: %v", kc.topics)

	for _, topic := range kc.topics {
		kc.wg.Add(1)
		go func(t string) {
			defer kc.wg.Done()

			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:        strings.Split(kc.brokers, ","),
				GroupID:        kc.groupID,
				Topic:          t,
				MinBytes:       10e3, // 10KB
				MaxBytes:       10e6, // 10MB
				CommitInterval: 1 * time.Second,
			})
			defer r.Close()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					m, err := r.ReadMessage(ctx)
					if err != nil {
						if ctx.Err() != nil {
							return
						}
						log.Printf("[KafkaConsumer Error] Gagal membaca pesan dari topik %s: %v", t, err)
						time.Sleep(5 * time.Second)
						continue
					}

					kc.processMessage(m.Value, t)
				}
			}
		}(topic)
	}

	return nil
}

func (kc *KafkaConsumer) Close() error {
	kc.wg.Wait()
	return nil
}

type GenericEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	AWB          string                 `json:"awb"`
	ResiID       string                 `json:"resi_id"`
	OccurredAt   interface{}            `json:"occurred_at"`
	Timestamp    interface{}            `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
	ActivityCode string                 `json:"activity_code"`
	LocationCode string                 `json:"location_code"`
}

func (kc *KafkaConsumer) processMessage(messageBytes []byte, topic string) {
	var event GenericEvent
	if err := json.Unmarshal(messageBytes, &event); err != nil {
		log.Printf("[KafkaConsumer Warning] Gagal unmarshal Kafka message pada topik %s: %v", topic, err)
		return
	}

	// 1. Resolve ResiID
	resiID := event.ResiID
	if resiID == "" {
		resiID = event.AWB
	}
	if resiID == "" {
		return // Abaikan jika tidak ada AWB/Resi
	}

	// 2. Resolve Activity Code
	activityCode := event.ActivityCode
	if activityCode == "" {
		switch event.EventType {
		case "order.created":
			activityCode = "CREATED"
		case "package.picked_up", "dispatch.assigned":
			activityCode = "ASSIGNED"
		case "package.in_transit":
			activityCode = "IN_TRANSIT"
		case "package.out_for_delivery":
			activityCode = "OUT_FOR_DELIVERY"
		case "package.delivered":
			activityCode = "DELIVERED"
		default:
			// Fallback: extract dari event_type
			if strings.HasPrefix(event.EventType, "package.") {
				activityCode = strings.ToUpper(strings.TrimPrefix(event.EventType, "package."))
			} else if strings.HasPrefix(event.EventType, "order.") {
				activityCode = strings.ToUpper(strings.TrimPrefix(event.EventType, "order."))
			} else {
				activityCode = strings.ToUpper(event.EventType)
			}
		}
	}

	if activityCode == "" {
		activityCode = "IN_TRANSIT"
	}

	// 3. Resolve Location Code
	locationCode := event.LocationCode
	if locationCode == "" {
		if val, ok := event.Metadata["location"]; ok {
			locationCode = val.(string)
		} else if val, ok := event.Metadata["sender_city"]; ok {
			locationCode = val.(string)
		} else if _, ok := event.Metadata["route_instruction"]; ok {
			locationCode = "ROUTE"
		} else {
			locationCode = "HUB" // Fallback default
		}
	}
	// Bersihkan prefix "Warehouse "
	if strings.HasPrefix(locationCode, "Warehouse ") {
		locationCode = strings.TrimPrefix(locationCode, "Warehouse ")
	}

	// 4. Resolve Timestamp
	var ts time.Time
	if event.OccurredAt != nil {
		ts = parseTime(event.OccurredAt)
	} else if event.Timestamp != nil {
		ts = parseTime(event.Timestamp)
	} else {
		ts = time.Now()
	}

	// 5. Cek Idempotensi (Duplikasi) sebelum simpan
	if kc.trackRepo != nil {
		history, err := kc.trackRepo.GetResiHistory(resiID)
		if err == nil && history != nil {
			for _, item := range history.History {
				if item.ActivityCode == activityCode && item.LocationCode == locationCode {
					log.Printf("[KafkaConsumer Idempotency] Event untuk resi %s dengan status %s di %s sudah tercatat. Diabaikan.", resiID, activityCode, locationCode)
					return
				}
			}
		}
	}

	trackingLog := model.TrackingLog{
		ResiID:       resiID,
		LocationCode: locationCode,
		ActivityCode: activityCode,
		Timestamp:    ts,
	}

	err := kc.repo.InsertLog(trackingLog)
	if err != nil {
		log.Printf("[KafkaConsumer Error] Gagal menyimpan log perjalanan untuk resi %s: %v", resiID, err)
		return
	}

	log.Printf("[KafkaConsumer Success] Berhasil mencatat status %s untuk resi %s di %s", activityCode, resiID, locationCode)
}

func parseTime(val interface{}) time.Time {
	if val == nil {
		return time.Now()
	}
	switch v := val.(type) {
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t
		}
		t, err = time.Parse("2006-01-02 15:04:05", v)
		if err == nil {
			return t
		}
		t, err = time.Parse("2006-01-02T15:04:05.000Z", v)
		if err == nil {
			return t
		}
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
	}
	return time.Now()
}
