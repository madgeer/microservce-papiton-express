# Tracking & Log Event Service
Layanan ini bertanggung jawab atas **pencatatan log aktivitas paket secara masif** (*write-heavy*) dan **penyediaan riwayat pelacakan pengiriman** (*read-heavy*) secara efisien untuk pembeli maupun kurir.

Layanan ini dirancang menggunakan **Clean Architecture** (pemisahan layer *handler*, *service*, *repository*, dan *model*) serta ditulis dalam bahasa pemrograman **Go**.

---

## ⚡ Arsitektur & Teknologi NoSQL (MongoDB)
Untuk menangani ribuan scan logistik per detik secara cepat, layanan ini menggunakan database berorientasi dokumen **MongoDB** yang sangat cepat untuk penulisan data skala besar:
* **Collection `tracking_logs`**: Menyimpan riwayat scan berupa koordinat transit, status, waktu pemindaian, dan tautan bukti foto.

```
tracking-and-logevent-service/
├── cmd/
│   └── main.go                  # Entrypoint utama (HTTP server & routing MongoDB)
├── internal/
│   ├── domain/                  # Kontrak interface
│   ├── handler/                 # HTTP controllers (API tracking handler & Event consumer)
│   ├── model/                   # Data transfer objects (BSON/JSON schemas)
│   ├── repository/              # Data access layer (MongoDB implementation)
│   └── service/                 # Core business logic (log processing and retrieval)
├── mocks/                       # Mocking repository untuk keperluan unit testing
└── k8s/                         # Konfigurasi Kubernetes Deployment
```

---

## 📥 Alur Kerja Pencatatan & Pembacaan Histori (Read/Write)

Layanan ini melayani pencatatan aktivitas paket melalui event-driven broker (Kafka) maupun REST API, dan menyajikan histori pelacakan paket.

```mermaid
sequenceDiagram
    autonumber
    actor Client as "Client / Customer UI"
    participant H as TrackingAPIHandler
    participant S as TrackingService
    participant R as MongoTrackingRepo
    database DB as MongoDB

    Note over Client, DB: Fase: Pembacaan Histori Pelacakan (Read)
    Client->>H: GET /api/v1/tracking?resi_id=RESI-123
    activate H
    H->>S: GetHistory("RESI-123")
    activate S
    S->>R: GetResiHistory("RESI-123")
    activate R
    R->>DB: Find({resi_id: "RESI-123"})
    activate DB
    DB-->>R: List of Activity Documents
    deactivate DB
    R-->>S: Return TrackingHistory, nil
    deactivate R
    S-->>H: Return TrackingHistory, nil
    deactivate S
    H-->>Client: 200 OK {"resi_id": "RESI-123", "history": [...]}
    deactivate H
```

---

## 🛠️ Uji Coba Unit Testing

Layanan ini dilengkapi dengan pengujian murni unit test menggunakan *interface mocking* (`gomock`).

Untuk menjalankan pengujian unit secara mandiri:
```bash
go test -v ./...
```
