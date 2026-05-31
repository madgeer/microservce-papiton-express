# PAPITON Express — Notification & Messaging Service

Service ini bertanggung jawab atas pengiriman notifikasi kepada pengguna
berdasarkan event yang dikonsumsi dari Kafka (order, shipping, tracking).

---

## Struktur Project

```
notification-service/
├── cmd/
│   └── main.go                        # Entry point aplikasi
├── internal/
│   ├── consumer/
│   │   └── kafka_consumer.go          # Konsumsi event dari Kafka
│   ├── processor/
│   │   ├── message_processor.go       # Business logic pemrosesan event
│   │   └── message_processor_test.go  # ✅ Unit test processor
│   ├── dispatcher/
│   │   ├── dispatcher.go              # Routing ke provider (Email/Push)
│   │   └── dispatcher_test.go         # ✅ Unit test dispatcher (dengan mock)
│   ├── provider/
│   │   └── provider.go                # Stub Email & Push provider
│   ├── repository/
│   │   └── notification_repo.go       # Penyimpanan log notifikasi
│   └── model/
│       └── event.go                   # Struct/model data
├── mocks/
│   └── mock_dispatcher.go             # Mock auto-generated (gomock)
├── tests/
│   └── functional/
│       └── notification_functional_test.go  # ✅ Functional test
├── docker-compose.test.yml            # Infrastruktur testing (Kafka + PostgreSQL)
├── Makefile                           # Perintah build & test
├── Jenkinsfile                        # CI/CD pipeline
└── go.mod
```

---

## Cara Menjalankan Test

### Prasyarat
```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Install dependencies
go mod tidy
```

### Unit Test (tanpa infrastruktur eksternal)
```bash
make test-unit

# Atau langsung:
go test ./internal/... -v -short -race -cover
```

### Functional Test (butuh Docker)
```bash
# Pastikan Docker berjalan, lalu:
make test-functional

# Atau langsung:
docker-compose -f docker-compose.test.yml up -d
go test ./tests/functional/... -v -timeout 60s
docker-compose -f docker-compose.test.yml down
```

### Semua Test
```bash
make test-all
```

---

## CI/CD Pipeline (Jenkins)

Urutan stage pipeline:
1. **Checkout** — Clone repository
2. **Unit Tests** — `go test ./internal/... -short` (cepat, tanpa infrastruktur)
3. **Lint / Vet** — `go vet ./...`
4. **Build Image** — Build Docker image
5. **Functional Tests** — Test dengan Kafka & PostgreSQL nyata
6. **Push Image** — Hanya di branch `main`
7. **Deploy ke Kubernetes** — Hanya di branch `main`
8. **Verify Deployment** — Cek rollout status

---

## Arsitektur Alur Data

```
Kafka Topics                  Notification Service
(papiton.events.*)
        │
        ▼
 [KafkaConsumer]
        │
        ▼
 [MessageProcessor]  ← Pure logic, mudah di-unit test
        │
        ▼
   [Dispatcher]      ← Routing berdasarkan Channel
     │       │
     ▼       ▼
 [Email]  [Push]     ← Provider (stub, bisa diganti implementasi nyata)
           │
           ▼
    [Repository]     ← Simpan log ke DB
```

---

## Catatan Teknis

- **Idempotency**: Simpan `event_id` yang sudah diproses untuk mencegah duplikasi notifikasi
- **Dead Letter Queue (DLQ)**: Tambahkan DLQ di Kafka untuk event yang gagal berulang kali
- **Retry Strategy**: Implementasikan exponential backoff sebelum event masuk DLQ
- **Rate Limiting**: Batasi frekuensi notifikasi per user untuk menghindari spam

---

## Environment Variables

| Variable        | Default              | Keterangan                    |
|-----------------|----------------------|-------------------------------|
| `KAFKA_BROKER`  | `localhost:9092`     | Alamat Kafka broker           |
| `KAFKA_GROUP_ID`| `notification-service-group` | Consumer group ID   |
| `FCM_SERVER_KEY`| -                    | Firebase Cloud Messaging key  |
| `SMTP_HOST`     | `smtp.papiton.id`    | SMTP server untuk email       |
| `FROM_EMAIL`    | `noreply@papiton.id` | Alamat email pengirim         |

---

Dikembangkan oleh **Kelompok 5** — PAPITON Express
