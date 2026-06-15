# Dokumentasi Alur Order & Tariff Service
**Layanan Pembuatan Order & Kalkulasi Tarif**

Service ini bertanggung jawab untuk melayani pembuatan order pengiriman barang baru, melakukan estimasi waktu pengiriman (ETA), dan menghitung ongkos kirim (tarif) berdasarkan jarak koordinat GPS (Haversine).

---

## 1. Spesifikasi Teknis & Database
*   **Port Layanan**: `8082` (Container) ➔ `8082` (Host)
*   **Penyimpanan**: PostgreSQL database (`papiton_order_tariff_service_db`)
*   **Tabel Database**: `orders`
*   **Event Broker**: Apache Kafka (Topik: `papiton.events.order`)

---

## 2. API Endpoints
*   `POST /api/v1/orders` : Membuat order pengiriman baru.
*   `GET /api/v1/orders/get` : Mengambil seluruh daftar order.
*   `GET /api/v1/orders/get?awb=XXX` : Mengambil detail order spesifik berdasarkan nomor resi (AWB).
*   `POST /api/v1/tariff/calculate` : Kalkulator ongkir mandiri tanpa menyimpan data (untuk simulasi).

---

## 3. Diagram Alur Kerja (Sequence Diagram)

```mermaid
sequenceDiagram
    autonumber
    actor Client as Pengguna / UI
    participant Handler as OrderHandler
    participant Service as OrderService
    participant Repo as OrderRepository (Postgres)
    participant Kafka as Kafka Publisher
    
    Client->>Handler: POST /api/v1/orders (Payload Sender/Recipient/Package)
    Handler->>Service: CreateOrder(req)
    
    Service->>Repo: GetDistance(senderGPS, recipientGPS)
    Note over Service,Repo: Perhitungan Jarak Menggunakan Formula Haversine
    Repo-->>Service: distance (km)
    
    Service->>Service: hitungTotalTarif(package, distance)
    Service->>Service: GenerateAWB(cityCode)
    Service->>Service: hitungETA(serviceType, distance)
    
    Service->>Repo: SaveOrder(req, res)
    Repo-->>Service: nil (Success)
    
    rect rgb(240, 240, 240)
        Note over Service,Kafka: Asynchronous Event Publishing
        Service->>Kafka: PublishOrderCreated(event)
    end
    
    Service-->>Handler: OrderResponse (AWB, Tariff, Distance, ETA)
    Handler-->>Client: JSON Response (HTTP 201 Created)
```
