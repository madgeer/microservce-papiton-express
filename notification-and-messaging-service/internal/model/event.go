package model

import "time"

// EventType mendefinisikan jenis event yang datang dari service lain
type EventType string

const (
	EventOrderCreated     EventType = "order.created"
	EventPackagePickedUp  EventType = "package.picked_up"
	EventPackageInTransit EventType = "package.in_transit"
	EventPackageDelivered EventType = "package.delivered"
	EventPackageFailed    EventType = "package.delivery_failed"
)

// IncomingEvent adalah struktur event yang dikonsumsi dari Kafka
type IncomingEvent struct {
	EventID    string            `json:"event_id"`
	EventType  EventType         `json:"event_type"`
	UserID     string            `json:"user_id"`
	AWB        string            `json:"awb"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// NotificationMessage adalah pesan yang siap dikirim ke user
type NotificationMessage struct {
	UserID  string
	Channel Channel
	Subject string
	Body    string
	AWB     string
}

// Channel mendefinisikan kanal pengiriman notifikasi
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

// NotificationStatus hasil pengiriman notifikasi
type NotificationStatus struct {
	Success bool
	Error   error
}
