package repository

import (
	"context"
	"database/sql"
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
	db *sql.DB
}

func NewPostgresNotificationRepository(db *sql.DB) *PostgresNotificationRepository {
	// Pastikan tabel notification_logs dibuat
	queryCreateTable := `
	CREATE TABLE IF NOT EXISTS notification_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id VARCHAR(50),
		awb VARCHAR(50) NOT NULL,
		channel VARCHAR(20) NOT NULL,
		subject TEXT,
		body TEXT,
		success BOOLEAN NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(queryCreateTable)
	if err != nil {
		log.Printf("[Repository] Gagal membuat tabel notification_logs: %v", err)
	}

	// Pastikan tabel processed_events dibuat untuk idempotensi
	queryCreateTableProcessed := `
	CREATE TABLE IF NOT EXISTS processed_events (
		event_id VARCHAR(100) PRIMARY KEY,
		processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(queryCreateTableProcessed)
	if err != nil {
		log.Printf("[Repository] Gagal membuat tabel processed_events: %v", err)
	}

	return &PostgresNotificationRepository{db: db}
}

func (r *PostgresNotificationRepository) SaveLog(
	ctx context.Context,
	msg model.NotificationMessage,
	success bool,
) error {
	queryInsert := `
	INSERT INTO notification_logs (user_id, awb, channel, subject, body, success, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	_, err := r.db.ExecContext(ctx, queryInsert, msg.UserID, msg.AWB, string(msg.Channel), msg.Subject, msg.Body, success)
	if err != nil {
		log.Printf("[Repository] Gagal menyimpan log ke DB: %v", err)
		return err
	}

	log.Printf(
		"[Repository] Menyimpan log | UserID: %s | AWB: %s | Channel: %s | Sukses: %v",
		msg.UserID, msg.AWB, msg.Channel, success,
	)
	return nil
}

func (r *PostgresNotificationRepository) GetLogs(ctx context.Context) ([]NotificationLog, error) {
	querySelect := `SELECT id, user_id, awb, channel, subject, body, success, created_at FROM notification_logs ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.QueryContext(ctx, querySelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []NotificationLog
	for rows.Next() {
		var l NotificationLog
		err := rows.Scan(&l.ID, &l.UserID, &l.AWB, &l.Channel, &l.Subject, &l.Body, &l.Success, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	if list == nil {
		list = []NotificationLog{}
	}
	return list, nil
}

func (r *PostgresNotificationRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&exists)
	if err != nil {
		log.Printf("[Repository] Gagal mengecek event_id %s: %v", eventID, err)
		return false, err
	}
	return exists, nil
}

func (r *PostgresNotificationRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	if eventID == "" {
		return nil
	}
	query := `INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT (event_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, eventID)
	if err != nil {
		log.Printf("[Repository] Gagal menandai event_id %s terproses: %v", eventID, err)
		return err
	}
	return nil
}

// InMemoryNotificationRepository adalah implementasi in-memory untuk testing
// Berguna untuk functional test yang tidak memerlukan database nyata
type InMemoryNotificationRepository struct {
	Logs            []NotificationLog
	ProcessedEvents map[string]bool
}

func NewInMemoryNotificationRepository() *InMemoryNotificationRepository {
	return &InMemoryNotificationRepository{
		ProcessedEvents: make(map[string]bool),
	}
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

func (r *InMemoryNotificationRepository) GetLogs(ctx context.Context) ([]NotificationLog, error) {
	return r.Logs, nil
}

func (r *InMemoryNotificationRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	if r.ProcessedEvents == nil {
		r.ProcessedEvents = make(map[string]bool)
	}
	return r.ProcessedEvents[eventID], nil
}

func (r *InMemoryNotificationRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	if eventID == "" {
		return nil
	}
	if r.ProcessedEvents == nil {
		r.ProcessedEvents = make(map[string]bool)
	}
	r.ProcessedEvents[eventID] = true
	return nil
}
