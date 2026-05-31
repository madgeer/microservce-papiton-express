package repository

import (
	"context"
	"log"
	"time"

	"papiton/notification-service/internal/model"
)

// NotificationLog adalah model tabel di database
type NotificationLog struct {
	ID        int64
	UserID    string
	AWB       string
	Channel   string
	Subject   string
	Body      string
	Success   bool
	CreatedAt time.Time
}

// PostgresNotificationRepository menyimpan log notifikasi ke PostgreSQL
type PostgresNotificationRepository struct {
	// db *sql.DB  // Uncomment dan isi saat implementasi nyata
}

func NewPostgresNotificationRepository( /* db *sql.DB */ ) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{}
}

func (r *PostgresNotificationRepository) SaveLog(
	ctx context.Context,
	msg model.NotificationMessage,
	success bool,
) error {
	// TODO: Implementasi query INSERT ke tabel notification_logs
	// Contoh query:
	// INSERT INTO notification_logs (user_id, awb, channel, subject, body, success, created_at)
	// VALUES ($1, $2, $3, $4, $5, $6, NOW())

	log.Printf(
		"[Repository] Menyimpan log | UserID: %s | AWB: %s | Channel: %s | Sukses: %v",
		msg.UserID, msg.AWB, msg.Channel, success,
	)
	return nil // Stub: selalu berhasil
}

// InMemoryNotificationRepository adalah implementasi in-memory untuk testing
// Berguna untuk functional test yang tidak memerlukan database nyata
type InMemoryNotificationRepository struct {
	Logs []NotificationLog
}

func NewInMemoryNotificationRepository() *InMemoryNotificationRepository {
	return &InMemoryNotificationRepository{}
}

func (r *InMemoryNotificationRepository) SaveLog(
	ctx context.Context,
	msg model.NotificationMessage,
	success bool,
) error {
	r.Logs = append(r.Logs, NotificationLog{
		UserID:    msg.UserID,
		AWB:       msg.AWB,
		Channel:   string(msg.Channel),
		Subject:   msg.Subject,
		Body:      msg.Body,
		Success:   success,
		CreatedAt: time.Now(),
	})
	return nil
}
