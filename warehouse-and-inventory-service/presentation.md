# 🏢 Warehouse & Inventory Service — Panduan Presentasi
**Direktori: `/warehouse-and-inventory-service`**

Layanan Go transaksional untuk mencatat barang masuk transit gudang (*inbound*), sorting prioritasi paket, dan manifest truk kontainer pengiriman.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Go (Golang) menggunakan `net/http` standard library.
*   **Database**: PostgreSQL (`papiton_warehouse`), mengelola tabel `warehouses`, `inbound_packages`, `manifests`, `manifest_packages`, dan `sorting_lanes`.
*   **Kafka Event**: Publisher untuk mengirimkan event `package.in_transit` ke topik `papiton.events.tracking` setiap kali paket sukses diproses masuk ke hub transit.

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Otentikasi API Key**: Middleware `requireAPIKey` mengamankan endpoint inbound fisik dan manifest agar tidak dapat dimanipulasi pihak luar.
2.  **Rate Limiter**: Dilindungi oleh middleware `withRateLimit` (100 RPM per IP).
3.  **Correlation ID**: Mengalirkan header `X-Correlation-ID` untuk audit pelacakan manifest.
4.  **Startup Fail-Fast**: Menguji konektivitas database menggunakan `db.Ping()` saat aplikasi pertama kali menyala, langsung mati (`log.Fatalf`) jika database bermasalah.

---

## 🔄 Integrasi Kafka (Payload Ter-enrich)
Event `package.in_transit` yang diterbitkan memuat informasi nama gudang tempat paket dipindai dan status aktivitas saat ini (`inventory.inbound`). Informasi ini langsung dikonsumsi oleh Tracking Service untuk menampilkan history resi terupdate.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Buka tab "Warehouse Scan"** pada Dashboard UI.
2.  Masukkan nomor AWB aktif, pilih ID Gudang transit (contoh: `WH-JKT`), lalu klik **Proses Scan Inbound**.
3.  Sistem akan mendeteksi status inbound, menentukan *sorting lane* (jenis regular/express), menyimpannya di database, dan memicu event Kafka.
4.  *Penjelasan Teknis*: Request melewati API Gateway `/api/proxy/inbound` ➔ diteruskan ke microservice port `8080`. Sistem merekam kedatangan paket fisik di tabel `inbound_packages` dan mengirim pesan asinkron ke topik Kafka `papiton.events.tracking`.
