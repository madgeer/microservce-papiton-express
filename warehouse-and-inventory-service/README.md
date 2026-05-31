# Warehouse & Inventory Service
Layanan ini bertanggung jawab mengelola operasional pergudangan (*Hub*), pemrosesan paket masuk (*inbound*), penyortiran paket (*sorting*), serta pengelompokan paket ke dalam kontainer truk keberangkatan (*manifest*).

Layanan ini dirancang menggunakan **Clean Architecture** (pemisahan layer *handler*, *service*, *repository*, dan *model*) serta ditulis dalam bahasa pemrograman **Go**.

---

## ⚡ Arsitektur & Struktur Direktori

```
warehouse-and-inventory-service/
├── cmd/
│   └── main.go                  # Entrypoint utama aplikasi (HTTP server & routing)
├── internal/
│   ├── handler/                 # HTTP controllers (inbound & manifest handlers)
│   ├── model/                   # Data transfer objects (DTO) & request/response structs
│   ├── repository/              # Data access layer (PostgreSQL implementation)
│   └── service/                 # Core business logic
├── migrations/                  # File migrasi database PostgreSQL
├── mocks/                       # Mocking repository untuk keperluan unit testing
└── test/
    └── helpers/                 # Test helpers & db container setup
```

---

## 📥 1. Inbound Flow (Penerimaan Paket)

Dipicu saat paket pertama kali discan di gudang/hub asal untuk mengubah status logistik menjadi `AT_HUB`.

```mermaid
sequenceDiagram
    autonumber
    actor Client as "Client / Scanner UI"
    participant H as InboundHandler
    participant S as InboundService
    participant R as InboundRepository
    participant DB as PostgreSQL DB

    Client->>H: POST /api/v1/inbound {"resi": "RESI-123", "warehouse_id": "WH-BDO"}
    activate H
    H->>H: Decode JSON Payload
    H->>S: ProcessInbound("RESI-123", "WH-BDO")
    activate S
    
    S->>R: UpdateStockStatus("RESI-123", "WH-BDO", "AT_HUB")
    activate R
    R->>DB: UPDATE inbound_packages SET status = 'AT_HUB' WHERE resi = 'RESI-123'
    activate DB
    DB-->>R: Row Affected (Success)
    deactivate DB
    R-->>S: Return nil (Success)
    deactivate R

    S-->>H: Return nil (Success)
    deactivate S
    
    H-->>Client: 200 OK {"status": 200, "message": "Inbound paket berhasil diproses", "resi": "RESI-123"}
    deactivate H
```

---

## 🚚 2. Outbound / Manifest Flow (Pemberangkatan Truk)

Alur penyusunan manifest untuk memasukkan paket ke dalam truk logistik dan merilis keberangkatannya.

```mermaid
sequenceDiagram
    autonumber
    actor Client as "Client / Dispatcher UI"
    participant H as ManifestHandler
    participant S as ManifestService
    participant R as ManifestRepository
    participant DB as PostgreSQL DB

    Note over Client, DB: Fase 1: Membuat Dokumen Manifest Baru
    Client->>H: POST /api/v1/manifest/create {"truck_id": "TRK-01", "driver_name": "Budi"}
    activate H
    H->>S: CreateNewManifest("TRK-01", "Budi")
    activate S
    S->>R: CreateManifest("TRK-01", "Budi")
    activate R
    R->>DB: INSERT INTO manifests (manifest_id, truck_id, driver_name, status) VALUES (...) RETURNING manifest_id
    activate DB
    DB-->>R: Return ManifestID (e.g. "MNF-001")
    deactivate DB
    R-->>S: Return "MNF-001", nil
    deactivate R
    S-->>H: Return "MNF-001", nil
    deactivate S
    H-->>Client: 200 OK {"status": 200, "manifest_id": "MNF-001", "message": "Manifest berhasil dibuat"}
    deactivate H

    Note over Client, DB: Fase 2: Memasukkan Paket ke dalam Manifest
    Client->>H: POST /api/v1/manifest/add {"manifest_id": "MNF-001", "resi": "RESI-123"}
    activate H
    H->>S: AddToManifest("MNF-001", "RESI-123")
    activate S
    S->>R: AddPackageToManifest("MNF-001", "RESI-123")
    activate R
    R->>DB: INSERT INTO manifest_packages (manifest_id, resi) VALUES ("MNF-001", "RESI-123")
    activate DB
    DB-->>R: Success
    deactivate DB
    R-->>S: Return nil
    deactivate R
    S-->>H: Return nil
    deactivate S
    H-->>Client: 200 OK {"status": 200, "message": "Paket berhasil dimasukkan ke manifest"}
    deactivate H

    Note over Client, DB: Fase 3: Truk Berangkat (Depart Manifest)
    Client->>H: POST /api/v1/manifest/update {"manifest_id": "MNF-001"}
    activate H
    H->>S: DepartManifest("MNF-001")
    activate S
    S->>R: UpdateManifestStatus("MNF-001", "DEPARTED")
    activate R
    R->>DB: UPDATE manifests SET status = 'DEPARTED' WHERE manifest_id = 'MNF-001'
    activate DB
    DB-->>R: Success
    deactivate DB
    R-->>S: Return nil
    deactivate R
    S-->>H: Return nil
    deactivate S
    H-->>Client: 200 OK {"status": 200, "message": "Status manifest berhasil di-update"}
    deactivate H
```

---

## 🛠️ Uji Coba Unit Testing

Layanan ini dilengkapi dengan pengujian murni unit test menggunakan *interface mocking* (`gomock`) dan SQL Mock (`go-sqlmock`).

Untuk menjalankan pengujian unit secara mandiri:
```bash
go test -v ./...
```
