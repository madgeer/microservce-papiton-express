package provider

import (
	"context"
	"fmt"
	"log"

	"papiton/notification-service/internal/model"
)

// ─── Email Provider ───────────────────────────────────────────────────────────

// EmailProvider mengirim notifikasi lewat email (implementasi nyata/stub)
type EmailProvider struct {
	SMTPHost string
	SMTPPort int
	From     string
}

func NewEmailProvider(smtpHost string, smtpPort int, from string) *EmailProvider {
	return &EmailProvider{
		SMTPHost: smtpHost,
		SMTPPort: smtpPort,
		From:     from,
	}
}

func (e *EmailProvider) Send(ctx context.Context, msg model.NotificationMessage) error {
	// TODO: Implementasi pengiriman email nyata menggunakan net/smtp atau library pihak ketiga
	// Contoh: gomail, sendgrid-go, dll.
	log.Printf("[EmailProvider] Mengirim email ke user %s | Subjek: %s", msg.UserID, msg.Subject)
	fmt.Printf("  To     : %s\n", msg.UserID)
	fmt.Printf("  Subject: %s\n", msg.Subject)
	fmt.Printf("  Body   : %s\n", msg.Body)
	return nil // Stub: selalu berhasil
}

// ─── Push Notification Provider ──────────────────────────────────────────────

// PushProvider mengirim push notification (implementasi nyata/stub)
type PushProvider struct {
	FCMServerKey string
}

func NewPushProvider(fcmServerKey string) *PushProvider {
	return &PushProvider{FCMServerKey: fcmServerKey}
}

func (p *PushProvider) Send(ctx context.Context, msg model.NotificationMessage) error {
	// TODO: Implementasi pengiriman push notification menggunakan FCM / APNs
	log.Printf("[PushProvider] Mengirim push notification ke user %s | AWB: %s", msg.UserID, msg.AWB)
	fmt.Printf("  UserID: %s\n", msg.UserID)
	fmt.Printf("  Body  : %s\n", msg.Body)
	return nil // Stub: selalu berhasil
}
