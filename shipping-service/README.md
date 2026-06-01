# Shipping & Dispatch Service
Layanan ini bertanggung jawab atas **penjadwalan kurir pengiriman**, penugasan kurir otomatis (*Auto-Dispatch*), pencatatan status penjemputan barang (*Confirm Pick-Up*), serta pelacakan titik GPS koordinat kurir secara *real-time*.

Layanan ini dirancang menggunakan **Clean Architecture** (pemisahan layer *handler*, *service*, *repository*, dan *model*) serta ditulis dalam bahasa pemrograman **Go**.

---

## ⚡ Arsitektur & Jaringan Database Ganda
Layanan ini mengintegrasikan dua teknologi database yang berbeda untuk fungsi optimal:
1. **PostgreSQL**: Digunakan untuk menyimpan data profil kurir (`couriers`) dan data transaksi log perjalanan kurir (`dispatches`).
2. **MongoDB**: Digunakan untuk menyimpan data log koordinat GPS kurir secara *real-time* (`courier_locations`) dengan throughput baca/tulis yang tinggi.

```
shipping-service/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint utama (HTTP server & routing postgres/mongo)
├── internal/
│   ├── domain/                  # Entity, model data, dan kontrak interface
│   ├── handler/
│   │   └── http/                # HTTP controllers (dispatch handler)
│   ├── repository/
│   │   ├── postgres/            # Relational DB logic (courier and dispatch SQL queries)
│   │   └── mongo/               # Document DB logic (real-time location BSON queries)
│   └── service/                 # Core business logic (Auto-Dispatching and GPS tracking)
├── mocks/                       # Mocking repository untuk keperluan unit testing
└── k8s/                         # Konfigurasi Kubernetes Deployment
```

---

## 📥 Alur Kerja Auto-Dispatch (Penugasan Kurir)

Proses ini dijalankan setelah order dibuat agar sistem secara otomatis mencari kurir terdekat yang berstatus `AVAILABLE` di wilayah zona penjemputan.

```mermaid
sequenceDiagram
    autonumber
    actor Client as "Client / System Trigger"
    participant H as DispatchHandler
    participant S as DispatchService
    participant R_PG as postgres.CourierRepository
    participant R_DP as postgres.DispatchRepository
    database DB as PostgreSQL DB

    Client->>H: POST /dispatch {"order_id": "ORD-123", "pickup_zone": "Bandung"}
    activate H
    H->>H: Decode JSON Payload
    H->>S: AutoDispatchPickUp(ctx, "ORD-123", "Bandung")
    activate S
    
    S->>R_PG: GetAvailableByZone(ctx, "Bandung")
    activate R_PG
    R_PG->>DB: SELECT * FROM couriers WHERE zone = 'Bandung' AND status = 'AVAILABLE'
    activate DB
    DB-->>R_PG: List of Available Couriers
    deactivate DB
    R_PG-->>S: Return []Courier, nil
    deactivate R_PG
    
    S->>S: Select First Available Courier (e.g. "C-001")
    
    S->>R_PG: UpdateStatus(ctx, "C-001", "ON_DUTY")
    activate R_PG
    R_PG->>DB: UPDATE couriers SET status = 'ON_DUTY' WHERE id = 'C-001'
    activate DB
    DB-->>R_PG: Success
    deactivate DB
    R_PG-->>S: Success
    deactivate R_PG

    S->>R_DP: Create(ctx, DispatchPayload)
    activate R_DP
    R_DP->>DB: INSERT INTO dispatches (...)
    activate DB
    DB-->>R_DP: Success
    deactivate DB
    R_DP-->>S: Success
    deactivate R_DP

    S-->>H: Return DispatchRecord, nil
    deactivate S
    
    H-->>Client: 200 OK {"id": "DSP-ORD-123", "courier_id": "C-001", "status": "ASSIGNED", ...}
    deactivate H
```

---

## 🛠️ Uji Coba Unit Testing

Layanan ini dilengkapi dengan pengujian murni unit test menggunakan *interface mocking* (`gomock`).

Untuk menjalankan pengujian unit secara mandiri:
```bash
go test -v ./...
```
