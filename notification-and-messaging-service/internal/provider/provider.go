package provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"papiton/notification-service/internal/model"
)

// ─── Email Provider ───────────────────────────────────────────────────────────

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

	if strings.Contains(msg.Subject, "FAIL-TEST") || strings.Contains(msg.Body, "FAIL-TEST") {
		return fmt.Errorf("simulated SMTP delivery failure (FAIL-TEST trigger detected)")
	}

	if e.SMTPUser != "" && e.SMTPPass != "" {
		log.Printf("[EmailProvider] Mencoba mengirim email via SMTP Server %s:%d...", e.SMTPHost, e.SMTPPort)
		auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPass, e.SMTPHost)

		to := []string{msg.UserID}
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
		log.Printf("[EmailProvider Simulasi] To: %s | Subject: %s | Body: %s", msg.UserID, msg.Subject, msg.Body)
	}

	return nil
}

// ─── Push Notification Provider (FCM HTTP v1 API) ────────────────────────────

// serviceAccount adalah struktur dari Google Service Account JSON.
type serviceAccount struct {
	ProjectID    string `json:"project_id"`
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	TokenURI     string `json:"token_uri"`
}

// PushProvider mengirim push notification via Firebase Cloud Messaging HTTP v1 API.
type PushProvider struct {
	projectID          string
	serviceAccountJSON string
}

func NewPushProvider(projectID, serviceAccountJSON string) *PushProvider {
	return &PushProvider{
		projectID:          projectID,
		serviceAccountJSON: serviceAccountJSON,
	}
}

func (p *PushProvider) Send(ctx context.Context, msg model.NotificationMessage) error {
	log.Printf("[PushProvider] Mengirim push notification ke user %s | AWB: %s", msg.UserID, msg.AWB)

	if strings.Contains(msg.Subject, "FAIL-TEST") || strings.Contains(msg.Body, "FAIL-TEST") {
		return fmt.Errorf("simulated FCM push failure (FAIL-TEST trigger detected)")
	}

	if p.serviceAccountJSON == "" || p.projectID == "" {
		log.Printf("[PushProvider Simulasi] UserID: %s | Body: %s", msg.UserID, msg.Body)
		return nil
	}

	// 1. Parse service account JSON
	var sa serviceAccount
	if err := json.Unmarshal([]byte(p.serviceAccountJSON), &sa); err != nil {
		return fmt.Errorf("[PushProvider] gagal parse service account JSON: %w", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}

	// 2. Dapatkan OAuth2 access token via Service Account JWT
	accessToken, err := getGoogleAccessToken(ctx, sa)
	if err != nil {
		return fmt.Errorf("[PushProvider] gagal mendapatkan access token: %w", err)
	}

	// 3. Sanitasi UserID (email) menjadi FCM topic name yang valid
	// FCM hanya mengizinkan: huruf, angka, `-`, `_`, `.`, `%`, `~`
	topicName := sanitizeFCMTopic(msg.UserID)

	// 4. Kirim notifikasi via FCM HTTP v1 API
	fcmURL := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", p.projectID)
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"topic": topicName,
			"notification": map[string]string{
				"title": msg.Subject,
				"body":  msg.Body,
			},
			"data": map[string]string{
				"awb":     msg.AWB,
				"user_id": msg.UserID,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[PushProvider] gagal marshal FCM payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fcmURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("[PushProvider] gagal membuat FCM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[PushProvider] gagal menghubungi FCM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyResp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[PushProvider] FCM API status %d: %s", resp.StatusCode, string(bodyResp))
	}

	log.Printf("[PushProvider Success] Push notification terkirim ke topik FCM: %s", topicName)
	return nil
}

// sanitizeFCMTopic mengubah email menjadi FCM topic name yang valid.
// Contoh: "user@example.com" → "user_at_example_dot_com"
func sanitizeFCMTopic(email string) string {
	r := strings.NewReplacer(
		"@", "_at_",
		".", "_dot_",
		"+", "_plus_",
		"/", "_slash_",
	)
	return "user_" + r.Replace(email)
}

// ─── Google OAuth2 via Service Account JWT (stdlib only) ─────────────────────

// getGoogleAccessToken mengambil OAuth2 access token dari Google menggunakan
// Service Account JWT — tanpa library eksternal.
func getGoogleAccessToken(ctx context.Context, sa serviceAccount) (string, error) {
	jwt, err := buildServiceAccountJWT(sa)
	if err != nil {
		return "", err
	}

	resp, err := http.PostForm(sa.TokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	})
	if err != nil {
		return "", fmt.Errorf("gagal request token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google token endpoint status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("gagal parse token response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("access_token kosong dari Google")
	}
	return result.AccessToken, nil
}

// buildServiceAccountJWT membuat JWT yang ditandatangani dengan private key service account.
func buildServiceAccountJWT(sa serviceAccount) (string, error) {
	now := time.Now()

	// Header
	headerJSON, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": sa.PrivateKeyID,
	})

	// Claims
	claimsJSON, _ := json.Marshal(map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"aud":   sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	claims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := header + "." + claims

	// Parse RSA private key dari PEM
	privKey, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("gagal parse private key: %w", err)
	}

	// Sign dengan SHA256withRSA
	h := sha256.New()
	h.Write([]byte(signingInput))
	digest := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("gagal sign JWT: %w", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseRSAPrivateKey mem-parse PEM private key dalam format PKCS8 atau PKCS1.
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	// Service account key dari Google menggunakan escape sequence \n
	pemStr = strings.ReplaceAll(pemStr, `\n`, "\n")

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("tidak dapat decode PEM block")
	}

	// Coba PKCS8 dulu (format Google Service Account)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key bukan RSA")
		}
		return rsaKey, nil
	}

	// Fallback ke PKCS1
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
