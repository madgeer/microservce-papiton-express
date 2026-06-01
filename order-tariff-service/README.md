# Order & Tariff Service
Layanan ini bertanggung jawab atas **manajemen order pengiriman baru** dan **perhitungan tarif pengiriman logistik secara presisi**.

Layanan ini dirancang menggunakan **Clean Architecture** (pemisahan layer *handler*, *service*, *repository*, dan *model*) serta ditulis dalam bahasa pemrograman **Go**.

---

## ⚡ Arsitektur & Struktur Direktori

```
order-tariff-service/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint utama aplikasi (HTTP server & routing)
├── internal/
│   ├── domain/                  # Entity, model data, dan kontrak interface
│   ├── handler/                 # HTTP controllers (order handler)
│   ├── repository/              # Data access layer (PostgreSQL & Redis implementation)
│   └── service/                 # Core business logic (AWB, ETA, and Tariff calculation)
├── mocks/                       # Mocking repository untuk keperluan unit testing
└── tests/                       # Functional/Integration testing
```

---

## 📥 Alur Bisnis Pembuatan Order

Proses ini dijalankan saat pelanggan melakukan booking pesanan kurir baru.

```mermaid
sequenceDiagram
    autonumber
    actor Client as "Client / Customer UI"
    participant H as OrderHandler
    participant S as OrderService
    participant R as OrderRepository
    database DB as PostgreSQL DB

    Client->>H: POST /api/v1/orders {"sender": {...}, "recipient": {...}, "package": {...}}
    activate H
    H->>H: Decode JSON Payload
    H->>S: CreateOrder(req)
    activate S
    
    S->>R: GetDistance(sender.Coordinate, recipient.Coordinate)
    activate R
    R-->>S: Return Distance (km)
    deactivate R
    
    S->>S: hitungTotalTarif(req, dist)
    S->>S: GenerateAWB(sender.City)
    S->>S: hitungETA(service_type, dist)
    
    S->>R: SaveOrder(req, res)
    activate R
    R->>DB: INSERT INTO orders (...)
    activate DB
    DB-->>R: Success
    deactivate DB
    R-->>S: Return nil
    deactivate R

    S-->>H: Return OrderResponse, nil
    deactivate S
    
    H-->>Client: 201 Created {"awb": "BDG240430120000X1Y2", "total": 25000, ...}
    deactivate H
```

---

## 🛠️ Uji Coba Unit Testing

Layanan ini dilengkapi dengan pengujian murni unit test menggunakan *interface mocking* (`gomock`).

Untuk menjalankan pengujian unit secara mandiri:
```bash
go test -v ./...
```
