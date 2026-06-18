package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"papiton/notification-service/internal/model"
)

// ─── Email Provider ───────────────────────────────────────────────────────────

// EmailProvider mengirim notifikasi lewat email (implementasi nyata/stub)
type EmailProvider struct {
	SMTPHost string
	SMTPPort int
	From     string
	SMTPUser string
	SMTPPass string
}

func NewEmailProvider(smtpHost string, smtpPort int, from string, smtpUser string, smtpPass string) *EmailProvider {
	return &EmailProvider{
		SMTPHost: smtpHost,
		SMTPPort: smtpPort,
		From:     from,
		SMTPUser: smtpUser,
		SMTPPass: smtpPass,
	}
}

func (e *EmailProvider) Send(ctx context.Context, msg model.NotificationMessage) error {
	log.Printf("[EmailProvider] Mengirim email ke user %s | Subjek: %s", msg.UserID, msg.Subject)
	fmt.Printf("  To     : %s\n", msg.UserID)
	fmt.Printf("  Subject: %s\n", msg.Subject)
	fmt.Printf("  Body   : %s\n", msg.Body)

	// Simulasi error untuk demo retry & DLQ
	if strings.Contains(msg.Subject, "FAIL-TEST") || strings.Contains(msg.Body, "FAIL-TEST") {
		return fmt.Errorf("simulated SMTP delivery failure (FAIL-TEST trigger detected)")
	}

	// Jika username dan password SMTP diisi, lakukan koneksi dan pengiriman SMTP riil
	if e.SMTPUser != "" && e.SMTPPass != "" {
		log.Printf("[EmailProvider] Mencoba mengirim email via SMTP Server %s:%d...", e.SMTPHost, e.SMTPPort)
		auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPass, e.SMTPHost)
		
		// Setup email headers & body
		to := []string{msg.UserID} // UserID diasumsikan alamat email di sistem
		headerSubject := fmt.Sprintf("Subject: %s\r\n", msg.Subject)
		headerMime := "MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n"
		emailBody := headerSubject + headerMime + msg.Body

		addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)
		err := smtp.SendMail(addr, auth, e.From, to, []byte(emailBody))
		if err != nil {
			log.Printf("[EmailProvider Error] Gagal mengirim email via SMTP: %v", err)
			return err
		}
		log.Printf("[EmailProvider Success] Email berhasil dikirim via SMTP ke %s", msg.UserID)
	} else {
		log.Println("[EmailProvider Info] SMTP_USER/SMTP_PASSWORD tidak dikonfigurasi. Berjalan dalam mode SIMULASI.")
	}

	return nil
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
	log.Printf("[PushProvider] Mengirim push notification ke user %s | AWB: %s", msg.UserID, msg.AWB)
	fmt.Printf("  UserID: %s\n", msg.UserID)
	fmt.Printf("  Body  : %s\n", msg.Body)

	// Simulasi error untuk demo retry & DLQ
	if strings.Contains(msg.Subject, "FAIL-TEST") || strings.Contains(msg.Body, "FAIL-TEST") {
		return fmt.Errorf("simulated FCM push failure (FAIL-TEST trigger detected)")
	}

	// Jika FCMServerKey dikonfigurasi, lakukan HTTP request riil ke Firebase API (Legacy HTTP protocol)
	if p.FCMServerKey != "" && p.FCMServerKey != "FCM_SERVER_KEY_PLACEHOLDER" && p.FCMServerKey != "FCM_KEY" && !strings.Contains(strings.ToUpper(p.FCMServerKey), "MOCK") {
		log.Println("[PushProvider] Mencoba mengirim push notification via Firebase API...")
		
		fcmURL := "https://fcm.googleapis.com/fcm/send"
		payload := map[string]interface{}{
			"to": "/topics/user_" + msg.UserID, // fallback sending to a topic corresponding to user
			"notification": map[string]string{
				"title": msg.Subject,
				"body":  msg.Body,
			},
			"data": map[string]string{
				"awb":     msg.AWB,
				"user_id": msg.UserID,
			},
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal FCM payload: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", fcmURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create FCM request: %w", err)
		}

		req.Header.Set("Authorization", "key="+p.FCMServerKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[PushProvider Error] Gagal menghubungi Firebase API: %v", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyResp, _ := io.ReadAll(resp.Body)
			log.Printf("[PushProvider Error] Firebase API merespon dengan status %d: %s", resp.StatusCode, string(bodyResp))
			return fmt.Errorf("FCM server returned status %d", resp.StatusCode)
		}

		log.Printf("[PushProvider Success] Push notification berhasil dikirim via Firebase API ke topik user_%s", msg.UserID)
	} else {
		log.Println("[PushProvider Info] FCM_SERVER_KEY tidak dikonfigurasi. Berjalan dalam mode SIMULASI.")
	}

	return nil
}
