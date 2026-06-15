# Dokumentasi Alur Shipping & Courier Service
**Layanan Manajemen Kurir & Auto-Dispatch**

Service ini mengelola pendaftaran kurir, status ketersediaan kurir, penugasan kurir secara otomatis berdasarkan kecocokan zona penjemputan (*Auto-Dispatch*), serta koordinat GPS real-time kurir.

---

## 1. Spesifikasi Teknis & Database
*   **Port Layanan**: `8080` (Container) ➔ `8081` (Host)
*   **Penyimpanan**:
    *   PostgreSQL database (`shipping_test_db`) untuk tabel `couriers` dan `dispatches`.
    *   MongoDB database (`shipping_db`) untuk data spasial koordinat GPS real-time kurir (`locations` collection).
*   **Event Broker**: Apache Kafka (Topik: `papiton.events.shipping` — Tipe Event: `package.picked_up`)

---

## 2. API Endpoints
*   `POST /dispatch` : Memilih kurir terdekat secara otomatis dan membuat log tugas (*dispatch*).
*   `POST /api/v1/couriers/register` : Mendaftarkan armada kurir baru.
*   `GET /api/v1/couriers` : Detail kurir (`?id=XXX`) atau daftar kurir per zona (`?zone=XXX`).
*   `PUT /api/v1/couriers/status` : Mengubah status ketersediaan kurir (*AVAILABLE, ON_DUTY, OFFLINE*).
*   `PUT /api/v1/couriers/location` : Mengirim koordinat GPS real-time kurir.
*   `POST /api/v1/dispatches/confirm` : Mengonfirmasi bahwa paket telah dijemput (*Confirm Pick-Up*).

---

## 3. Diagram Alur Kerja (Sequence Diagram)

```mermaid
sequenceDiagram
    autonumber
    participant OrderSvc as Order Service / Kafka
    participant Handler as DispatchHandler
    participant Service as DispatchService
    participant CRepo as CourierRepo (Postgres)
    participant DRepo as DispatchRepo (Postgres)
    participant Kafka as Kafka Publisher

    Note over OrderSvc,Handler: Dipicu oleh Kafka event order.created / Trigger HTTP
    OrderSvc->>Handler: POST /dispatch (OrderID, PickupZone)
    Handler->>Service: AutoDispatchPickUp(orderID, pickupZone)
    
    Service->>CRepo: GetAvailableByZone(pickupZone)
    CRepo-->>Service: List Available Couriers (Asep, Budi, etc.)
    
    alt Tidak Ada Kurir Tersedia
        Service-->>Handler: error ("tidak ada kurir tersedia")
        Handler-->>OrderSvc: HTTP 500 / Log Error
    else Kurir Tersedia (Pilih Kurir Pertama)
        Service->>CRepo: UpdateStatus(courierID, "ON_DUTY")
        Service->>DRepo: Create(DispatchInfo)
        Note over Service,DRepo: Simpan data tugas kurir dengan status ASSIGNED
        
        rect rgb(240, 240, 240)
            Service->>Kafka: PublishDispatchAssigned(event)
            Note over Service,Kafka: Topik: papiton.events.shipping (Tipe: package.picked_up)
        end
        
        Service-->>Handler: Dispatch Details (ID, CourierID, Route)
        Handler-->>OrderSvc: JSON Response (HTTP 200 OK)
    end
```
