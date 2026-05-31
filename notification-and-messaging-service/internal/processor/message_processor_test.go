package processor_test

import (
	"strings"
	"testing"
	"time"

	"papiton/notification-service/internal/model"
	"papiton/notification-service/internal/processor"
)

func TestMessageProcessor_Process_OrderCreated(t *testing.T) {
	p := processor.NewMessageProcessor()
	event := model.IncomingEvent{
		EventID:    "evt-001",
		EventType:  model.EventOrderCreated,
		UserID:     "user-123",
		AWB:        "PAPITON-2025-001",
		OccurredAt: time.Now(),
	}

	result, err := p.Process(event)

	if err != nil {
		t.Fatalf("tidak seharusnya ada error, tapi dapat: %v", err)
	}
	if result.UserID != "user-123" {
		t.Errorf("UserID salah. Harapan: user-123, Dapat: %s", result.UserID)
	}
	if result.AWB != "PAPITON-2025-001" {
		t.Errorf("AWB salah. Harapan: PAPITON-2025-001, Dapat: %s", result.AWB)
	}
	if result.Channel != model.ChannelPush {
		t.Errorf("Channel harusnya Push untuk OrderCreated, dapat: %s", result.Channel)
	}
	if result.Subject == "" {
		t.Error("Subject tidak boleh kosong")
	}
	if result.Body == "" {
		t.Error("Body tidak boleh kosong")
	}
}

func TestMessageProcessor_Process_PackageFailed_ShouldUseEmail(t *testing.T) {
	p := processor.NewMessageProcessor()
	event := model.IncomingEvent{
		EventType: model.EventPackageFailed,
		UserID:    "user-456",
		AWB:       "PAPITON-2025-002",
		Metadata:  map[string]string{"reason": "Penerima tidak ada di tempat"},
	}

	result, err := p.Process(event)

	if err != nil {
		t.Fatalf("tidak seharusnya ada error: %v", err)
	}
	if result.Channel != model.ChannelEmail {
		t.Errorf("PackageFailed harus pakai Email, dapat: %s", result.Channel)
	}
}

func TestMessageProcessor_Process_InTransit_ContainsLocation(t *testing.T) {
	p := processor.NewMessageProcessor()
	event := model.IncomingEvent{
		EventType: model.EventPackageInTransit,
		UserID:    "user-789",
		AWB:       "PAPITON-2025-003",
		Metadata:  map[string]string{"location": "Hub Jakarta Timur"},
	}

	result, err := p.Process(event)

	if err != nil {
		t.Fatalf("tidak seharusnya ada error: %v", err)
	}
	if !strings.Contains(result.Body, "Hub Jakarta Timur") {
		t.Errorf("Body harus mengandung nama lokasi, dapat: %s", result.Body)
	}
}

func TestMessageProcessor_Process_AllEventTypes(t *testing.T) {
	p := processor.NewMessageProcessor()

	testCases := []struct {
		name            string
		event           model.IncomingEvent
		expectedChannel model.Channel
		expectError     bool
	}{
		{
			name: "order created → push notification",
			event: model.IncomingEvent{
				EventType: model.EventOrderCreated,
				UserID:    "u1", AWB: "AWB001",
			},
			expectedChannel: model.ChannelPush,
			expectError:     false,
		},
		{
			name: "package picked up → push notification",
			event: model.IncomingEvent{
				EventType:  model.EventPackagePickedUp,
				UserID:     "u2",
				AWB:        "AWB002",
				OccurredAt: time.Now(),
			},
			expectedChannel: model.ChannelPush,
			expectError:     false,
		},
		{
			name: "package delivered → push notification",
			event: model.IncomingEvent{
				EventType: model.EventPackageDelivered,
				UserID:    "u3", AWB: "AWB003",
			},
			expectedChannel: model.ChannelPush,
			expectError:     false,
		},
		{
			name: "delivery failed → EMAIL (kritis!)",
			event: model.IncomingEvent{
				EventType: model.EventPackageFailed,
				UserID:    "u4", AWB: "AWB004",
				Metadata: map[string]string{"reason": "Alamat tidak ditemukan"},
			},
			expectedChannel: model.ChannelEmail,
			expectError:     false,
		},
		{
			name:        "event type tidak dikenal → error",
			event:       model.IncomingEvent{EventType: "unknown.event", UserID: "u5", AWB: "AWB005"},
			expectError: true,
		},
		{
			name:        "user_id kosong → error validasi",
			event:       model.IncomingEvent{EventType: model.EventOrderCreated, AWB: "AWB006"},
			expectError: true,
		},
		{
			name:        "AWB kosong → error validasi",
			event:       model.IncomingEvent{EventType: model.EventOrderCreated, UserID: "u6"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.Process(tc.event)

			if tc.expectError {
				if err == nil {
					t.Error("seharusnya ada error, tapi tidak ada")
				}
				return
			}

			if err != nil {
				t.Fatalf("tidak seharusnya error: %v", err)
			}
			if result.Channel != tc.expectedChannel {
				t.Errorf("channel salah. Harapan: %s, Dapat: %s",
					tc.expectedChannel, result.Channel)
			}
		})
	}
}
