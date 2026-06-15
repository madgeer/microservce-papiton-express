# 🔄 Aliran Data & Interaksi Antar-Service (PAPITON Express)

Dokumen ini menjelaskan aliran data (*data flow*) secara menyeluruh, baik interaksi operasional antar-microservice (OLTP) maupun aliran data analitis ke Data Warehouse (OLAP).

---

## 🗺️ 1. Diagram Integrasi Sistem (End-to-End System Flow)

Diagram di bawah ini menggambarkan bagaimana sebuah transaksi berjalan, mulai dari pemesanan oleh pelanggan, penugasan kurir, pelacakan gudang, pengiriman notifikasi, hingga penyerapan data secara real-time ke Data Warehouse (DWH).

```mermaid
flowchart TD
    %% Actors
    Customer([Pelanggan])
    CourierActor([Kurir])
    ScannerActor([Petugas Gudang])
    Manager([Manajemen / Owner])

    %% Microservices
    subgraph Microservices_OLTP [Layanan Operasional - OLTP]
        order_svc[Order & Tariff Service]
        shipping_svc[Shipping & Dispatch Service]
        warehouse_svc[Warehouse & Inventory Service]
        tracking_svc[Tracking & Log Service]
        notif_svc[Notification Service]
    end

    %% Databases
    subgraph Databases_OLTP [Penyimpanan Operasional]
        order_db[(Order DB: Postgres)]
        shipping_db[(Shipping DB: Postgres)]
        warehouse_db[(Warehouse DB: Postgres)]
        tracking_db[(Tracking DB: MongoDB)]
    end

    %% Event Broker
    subgraph Kafka_Broker [Message Broker]
        topic_order{{"papiton.events.order"}}
        topic_shipping{{"papiton.events.shipping"}}
        topic_tracking{{"papiton.events.tracking"}}
    end

    %% Data Engineering
    subgraph Data_Engineering_OLAP [Analitis - OLAP]
        etl_consumer[ETL Real-time Consumer]
        dwh_db[(Data Warehouse: Postgres)]
        dashboard[Dashboard Analitik / UI]
    end

    %% Flows - Step 1: Order
    Customer -->|1. Buat Order| order_svc
    order_svc -->|Simpan Transaksi| order_db
    order_svc -->|2. Publish order.created| topic_order

    %% Flows - Step 2: Shipping
    topic_order -.->|Trigger Auto-Dispatch| shipping_svc
    shipping_svc -->|Simpan Penugasan| shipping_db
    shipping_svc -->|3. Publish dispatch.assigned| topic_shipping

    %% Flows - Step 3: Warehouse & Tracking
    CourierActor -->|Serah Terima Paket| ScannerActor
    ScannerActor -->|4. Scan Inbound| warehouse_svc
    warehouse_svc -->|Simpan Manifest/Inbound| warehouse_db
    warehouse_svc -->|5. Publish package.transit| topic_tracking

    %% Flows - Step 4: Tracking & Notif
    topic_tracking -.->|Sinkronisasi Log| tracking_svc
    tracking_svc -->|Simpan Log Scan| tracking_db
    
    topic_order -.->|Kirim SMS/Email| notif_svc
    topic_shipping -.->|Kirim SMS/Email| notif_svc
    topic_tracking -.->|Kirim SMS/Email| notif_svc
    notif_svc -->|Kirim ke| Customer

    %% Flows - Step 5: ETL & Data Warehouse (DWH)
    topic_order ===>|Consume Event| etl_consumer
    topic_shipping ===>|Consume Event| etl_consumer
    topic_tracking ===>|Consume Event| etl_consumer

    etl_consumer -.->|Lookup Metadata| order_db
    etl_consumer -.->|Lookup Metadata| shipping_db
    etl_consumer -.->|Lookup Metadata| warehouse_db

    etl_consumer ===>|6. Load Star Schema| dwh_db
    dwh_db -->|7. Query Visualisasi| dashboard
    dashboard -->|Pantau Laporan| Manager
```

---

## 🔄 2. Detail Aliran Data Per-Siklus Transaksi

Berikut adalah penjelasan langkah demi langkah aliran data antar-service:

### A. Siklus Pembuatan Order (Order Creation Cycle)
1. **Pelanggan** membuat pesanan melalui UI dengan endpoint `POST /api/v1/orders`.
2. **Order Service** menghitung tarif pengiriman (menggunakan Redis Cache untuk jarak rute) dan menyimpan data order ke `order_db` dengan status `CREATED`.
3. **Order Service** mempublikasikan event `order.created` ke topik Kafka `papiton.events.order`.
4. **Notification Service** mendengarkan event ini dan mengirimkan email/push notifikasi konfirmasi pesanan ke Pelanggan.
5. **ETL Service** (Data Engineering) mendengarkan event ini:
   - Mengambil detail order dari `order_db`.
   - Mengisi dimensi `dim_location` (lokasi asal/tujuan), `dim_service`, dan `dim_date` di Data Warehouse.
   - Membuat baris baru di tabel fakta `fact_shipment` (dengan status kurir/courier_profile_key `-1` / Belum Ditugaskan).

### B. Siklus Penugasan Kurir (Courier Dispatch Cycle)
1. Event `order.created` di Kafka memicu **Shipping Service** untuk mencarikan kurir terdekat yang berstatus `AVAILABLE` di wilayah penjemputan.
2. **Shipping Service** menugaskan kurir tersebut, mengubah status kurir menjadi `ON_DUTY` di `shipping_db`, dan menyimpan data penugasan (*dispatch*).
3. **Shipping Service** mempublikasikan event `dispatch.assigned` ke topik Kafka `papiton.events.shipping`.
4. **ETL Service** mendengarkan event ini:
   - Mengambil data kurir dari `shipping_db`.
   - Memperbarui tabel dimensi `dim_courier_profile` di DWH.
   - Memperbarui `courier_profile_key` di tabel fakta `fact_shipment` untuk menghubungkan pesanan tersebut dengan profil kurir bersangkutan.

### C. Siklus Transit & Inbound Gudang (Warehouse & Transit Cycle)
1. Kurir menjemput paket dari pengirim dan membawanya ke Hub asal (Gudang Transit).
2. Petugas gudang melakukan scan barcode paket melalui API `POST /api/v1/inbound` di **Warehouse Service**.
3. **Warehouse Service** memperbarui status paket menjadi `AT_HUB` di `warehouse_db`.
4. **Warehouse Service** mempublikasikan event `package.inbound` / `package.transit` ke topik Kafka `papiton.events.tracking`.
5. **Tracking Service** mengonsumsi event ini dan menyimpannya sebagai log sejarah perjalanan paket di database NoSQL MongoDB (`tracking_db`).
6. **ETL Service** mendengarkan event ini:
   - Mengambil data gudang dari `warehouse_db` dan memperbarui `dim_warehouse` di DWH.
   - Memperbarui `warehouse_key` di tabel fakta `fact_shipment` sehingga sistem mengetahui keberadaan paket fisik saat ini.

---

## 📊 3. Matriks Komponen Integrasi

| Nama Topik Kafka | Tipe Event | Pengirim (Producer) | Penerima Operasional (Consumer) | Aksi ETL (Data Warehouse Load) |
| :--- | :--- | :--- | :--- | :--- |
| `papiton.events.order` | `order.created` | Order & Tariff Service | Notification Service | Membuat baris di `dim_location`, `dim_service`, dan `fact_shipment` (initial record). |
| `papiton.events.shipping` | `package.picked_up` / `dispatch.assigned` | Shipping & Dispatch Service | Notification Service | Mengisi profil kurir di `dim_courier_profile` dan memperbarui `courier_profile_key` di `fact_shipment`. |
| `papiton.events.tracking` | `package.in_transit` / `package.delivered` | Warehouse / Shipping Service | Tracking & Log Service, Notification Service | Mengisi informasi gudang di `dim_warehouse` dan meng-update lokasi transit di `fact_shipment`. |
