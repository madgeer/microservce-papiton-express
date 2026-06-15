# PAPITON Express - Rancangan Data Warehouse [REVISI]

**Sistem Mikroservis Logistik Terintegrasi**
**Tugas Besar Komputasi Awan - Kelompok 5 (2026)**

---

## 👥 Anggota Tim

* **Muhammad Rizkiana Pratama** - Warehouse & Inventory Service
* **Nadhif Arva Anargya** - Tracking & Log Event Service
* **Shidqi Rasyad Firjatullah** - Order & Tariff Service
* **Dicka Fachrunaldo K.** - Shipping & Dispatch Service
* **Muhammad Fittra Novria** - Notification & Messaging Service

---

## 🔄 Ringkasan Perubahan Utama (Revisi)

Dokumen ini merupakan revisi dari rancangan DWH awal berdasarkan masukan dosen/reviewer. Perubahan difokuskan pada perbaikan konseptual star schema agar sesuai dengan prinsip data warehouse (agregatif, bukan menyimpan data transaksional individu sebagai dimensi):

### Perbaikan Utama (Revisi Kedua)
*   **Pembersihan Dimensi Individual (User/Kurir/Transaksi)**: Menghapus `dim_order` dan `dim_courier`. Entitas individual (seperti nama kurir, ID kurir, nama pengirim) tidak dijadikan dimensi karena DWH berfokus pada analisis data agregat.
*   **Penerapan Degenerate Dimension**: Kolom `awb` (kode resi transaksi) dan `order_status` kini diletakkan langsung di dalam tabel fakta `fact_shipment` sebagai *degenerate dimension*.
*   **Dimensi Lokasi Terstruktur (`dim_location`)**: Mengganti kolom teks alamat bebas/kota string dengan dimensi wilayah hierarkis terstruktur (`province`, `city_kabupaten`, `district_kecamatan`, `subdistrict_kelurahan`) untuk menghindari free-text string yang bukan merupakan dimensi standar. Tabel fakta menggunakan role-playing dimension (`origin_location_key` dan `destination_location_key`).
*   **Dimensi Profil Kurir (`dim_courier_profile`)**: Menggantikan `dim_courier` individual dengan dimensi profil agregat (`zone`, `vehicle_type`).
*   **Eksplisitasi Rumus Agregasi Measure**: Menjelaskan secara rinci tipe agregasi (`SUM`, `COUNT`, `AVG`) untuk setiap metrik/measure yang disajikan pada laporan analitik.

---

# 1. 🌟 Rancangan Data Warehouse (Star Schema)

Proses identifikasi dilakukan dengan menganalisis metrik bisnis utama (Fakta) dan konteks analisis (Dimensi).

## Diagram Star Schema (Mermaid)

```mermaid
erDiagram
    fact_shipment {
        BIGSERIAL shipment_key PK
        VARCHAR awb "Degenerate Dimension"
        INT date_key FK
        INT origin_location_key FK "Role-playing: Origin"
        INT destination_location_key FK "Role-playing: Destination"
        INT courier_profile_key FK
        INT warehouse_key FK
        INT service_key FK
        VARCHAR order_status "Degenerate Dimension"
        NUMERIC tarif_total "Measure (Additive)"
        NUMERIC distance_km "Measure (Additive)"
        NUMERIC package_weight "Measure (Additive)"
        NUMERIC volumetric_weight "Measure (Additive)"
        INT notification_count "Measure (Additive)"
        TIMESTAMP etl_loaded_at
    }

    fact_notification {
        BIGSERIAL notif_key PK
        VARCHAR awb "Degenerate Dimension"
        INT date_key FK
        BIGINT notif_type_key FK
        BOOLEAN success "Measure (Boolean-to-Int)"
        TIMESTAMP notif_at
        TIMESTAMP etl_loaded_at
    }

    dim_location {
        INT location_key PK
        VARCHAR province
        VARCHAR city_kabupaten
        VARCHAR district_kecamatan
        VARCHAR subdistrict_kelurahan
    }

    dim_courier_profile {
        INT courier_profile_key PK
        VARCHAR zone
        VARCHAR vehicle_type
    }

    dim_date {
        INT date_key PK
        DATE full_date
        VARCHAR day_of_week
        SMALLINT month_num
        VARCHAR month_name
        SMALLINT quarter
        SMALLINT year
        BOOLEAN is_weekend
    }

    dim_service {
        INT service_key PK
        VARCHAR service_type
        BOOLEAN has_insurance
        BOOLEAN has_packing
        NUMERIC base_price_per_km
        NUMERIC insurance_fee
    }

    dim_warehouse {
        INT warehouse_key PK
        VARCHAR warehouse_id
        VARCHAR warehouse_name
        VARCHAR city
        VARCHAR region
        VARCHAR warehouse_type
    }

    dim_notification_type {
        BIGSERIAL notif_type_key PK
        VARCHAR channel
        VARCHAR event_type
    }

    dim_date ||--o{ fact_shipment : "date_key"
    dim_location ||--o{ fact_shipment : "origin_location_key"
    dim_location ||--o{ fact_shipment : "destination_location_key"
    dim_courier_profile ||--o{ fact_shipment : "courier_profile_key"
    dim_warehouse ||--o{ fact_shipment : "warehouse_key"
    dim_service ||--o{ fact_shipment : "service_key"

    dim_date ||--o{ fact_notification : "date_key"
    dim_notification_type ||--o{ fact_notification : "notif_type_key"
```

## Definisi Tabel Utama

### fact_shipment
Tabel fakta transaksi pengiriman. Butiran data (*grain*) adalah satu baris per resi pengiriman (AWB). Memiliki degenerate dimensions (`awb`, `order_status`) dan referensi ke dimensi geografis pengirim/penerima.
*   **Measures**:
    *   `tarif_total` (Bisa di-`SUM` untuk total omzet atau di-`AVG` untuk rata-rata tarif)
    *   `distance_km` (Bisa di-`SUM` untuk total jarak tempuh atau di-`AVG` untuk rata-rata jarak)
    *   `package_weight` (Bisa di-`AVG` untuk rata-rata berat paket)
    *   `volumetric_weight` (Bisa di-`AVG` untuk rata-rata berat volumetrik)
    *   `notification_count` (Bisa di-`SUM` untuk total notifikasi yang dikirimkan)
    *   `COUNT(shipment_key)` (Agregasi dinamis untuk menghitung jumlah total paket/transaksi)

### fact_notification
Tabel fakta untuk merekam pengiriman notifikasi dari sistem.
*   **Measures**:
    *   `success` (Bisa di-`SUM(CASE WHEN success THEN 1 ELSE 0 END)` untuk total notifikasi sukses, atau di-`AVG(CASE WHEN success THEN 1 ELSE 0 END)*100` untuk rasio persentase kesuksesan notifikasi).
    *   `COUNT(notif_key)` (Agregasi dinamis untuk menghitung jumlah total notifikasi yang dikirimkan)

### dim_location
Dimensi geografi terstruktur. Tidak ada alamat bebas (string acak), melainkan terbagi atas struktur administratif resmi: Provinsi (`province`), Kabupaten/Kota (`city_kabupaten`), Kecamatan (`district_kecamatan`), dan Kelurahan (`subdistrict_kelurahan`). Digunakan secara *role-playing* sebagai lokasi asal dan tujuan.

### dim_courier_profile
Dimensi profil/kelompok kurir. Tidak menyimpan nama atau ID kurir secara individual (yang merupakan data operasional OLTP), melainkan mengelompokkan data berdasarkan karakteristik agregat kurir yaitu zona wilayah (`zone`) dan jenis kendaraan (`vehicle_type`). Memiliki *sentinel row* `courier_profile_key = -1` (Zone: N/A, Vehicle: N/A) untuk status pengiriman "Belum Ditugaskan".

### dim_date
Dimensi waktu dengan format Primary Key `YYYYMMDD` untuk mempermudah kueri analisis berbasis rentang tanggal.

### dim_warehouse
Dimensi lokasi gudang hub/transit. Menyimpan properti administratif gudang (`warehouse_id`, `warehouse_name`, `city`, `region`, `warehouse_type`).

### dim_notification_type
Dimensi tipe dan saluran notifikasi, mengelompokkan berdasarkan `channel` (email, push) dan `event_type` (order.created, package.picked_up, dll).

---

# 2. ⚙️ Proses ETL (Extraction, Transformation, Load)

Data diekstraksi secara real-time melalui Kafka Event Consumer dan disimpan ke dalam PostgreSQL Data Warehouse.

## Matriks Sumber Data
*   **Order DB**: Tabel `orders` untuk memuat data pengiriman dasar, tarif, berat, dan dimensi geografis awal.
*   **Shipping DB**: Tabel `couriers` dan `dispatches` untuk memuat data penugasan kurir dan memetakan profil kurir.
*   **Warehouse DB**: Tabel `warehouses` dan `inbound_packages` untuk memetakan transit gudang.
*   **Kafka Events**: Sebagai trigger real-time untuk pemrosesan ETL.

## Aturan Transformasi (T1-T10)
*   **T1 - Konsolidasi Data**: Menggabungkan data order dengan data penugasan kurir dan inbound gudang berdasarkan AWB.
*   **T2 - Normalisasi Wilayah**: Memetakan teks kota (e.g. "Bandung") ke dalam baris dimensi terstruktur `dim_location` (Provinsi, Kabupaten, Kecamatan, Kelurahan) menggunakan standarisasi kamus wilayah.
*   **T3 - Perhitungan Berat Volumetrik**: `(length_cm * width_cm * height_cm) / 6000.0`
*   **T4 - Penanganan NULL**: Mengarahkan penugasan kurir kosong ke sentinel row `courier_profile_key = -1`.

---

# 3. 📊 Laporan dan Analitik (OLAP Metrics)

Berikut adalah definisi matrik laporan operasional yang dibangun di atas rancangan Star Schema ini:

| ID Laporan | Nama Laporan | Dimensi Analisis | Formula Agregasi (Measures) | Tujuan Bisnis |
| :--- | :--- | :--- | :--- | :--- |
| **L1** | **Volume & Revenue Pengiriman** | `dim_date.year`, `dim_date.month_name`, `dim_service.service_type` | <ul><li>`COUNT(fact_shipment.shipment_key)` (Jumlah Paket)</li><li>`SUM(fact_shipment.tarif_total)` (Total Pendapatan)</li><li>`AVG(fact_shipment.tarif_total)` (Rata-rata Tarif)</li></ul> | Mengukur performa pertumbuhan transaksi dan pendapatan bulanan per jenis layanan. |
| **L2** | **Performa Kurir per Zona** | `dim_courier_profile.zone`, `dim_courier_profile.vehicle_type` | <ul><li>`COUNT(fact_shipment.shipment_key)` (Jumlah Pengiriman)</li><li>`AVG(fact_shipment.distance_km)` (Rata-rata Jarak Tempuh)</li></ul> | Menganalisis beban kerja kurir berdasarkan wilayah operasional dan jenis kendaraan. |
| **L3** | **Analisis Kemacetan Hub (Warehouse)** | `dim_warehouse.warehouse_name`, `fact_shipment.order_status` | <ul><li>`COUNT(fact_shipment.shipment_key)` (Jumlah Paket Tertahan/Transit)</li></ul> | Mengidentifikasi bottleneck penumpukan barang pada gudang hub tertentu. |
| **L4** | **Penyebaran Geografis Pengiriman** | `dim_location.province` (Asal), `dim_location.city_kabupaten` (Tujuan) | <ul><li>`COUNT(fact_shipment.shipment_key)` (Jumlah Transaksi)</li><li>`SUM(fact_shipment.tarif_total)` (Total Omzet)</li></ul> | Mengetahui rute dan wilayah pengiriman dengan kepadatan volume tertinggi untuk alokasi resource. |
| **L5** | **Rasio Keberhasilan Notifikasi** | `dim_notification_type.channel`, `dim_notification_type.event_type` | <ul><li>`COUNT(fact_notification.notif_key)` (Total Notifikasi Terkirim)</li><li>`AVG(fact_notification.success::INT) * 100` (Persentase Keberhasilan)</li></ul> | Mengukur kualitas pengiriman notifikasi SMS/Email ke customer. |
