# 📦 Order & Tariff Service — Panduan Presentasi
**Direktori: `/order-tariff-service`**

Layanan Go transaksional untuk memproses pemesanan baru, verifikasi koordinat, dan kalkulasi tarif berdasarkan jarak.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Go (Golang) menggunakan `net/http` standard library.
*   **Database**: PostgreSQL (`papiton_order_tariff_service_db`), tabel `orders` untuk menyimpan transaksi order.
*   **Caching**: Redis (`order-redis:6379`) untuk menyimpan cache kalkulasi jarak koordinat antar-kota (TTL 24 jam) guna memangkas latensi API request berulang.
*   **Kafka Event**: Publisher untuk mempublikasikan event `order.created` ke topik `papiton.events.order`.

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Otentikasi API Key**: Middleware `requireAPIKey` menolak request tanpa header `X-API-Key` yang cocok dengan environment (`dev-papiton-api-key-2024`).
2.  **Rate Limiter**: Menggunakan in-memory map IP untuk membatasi request maksimal 100 RPM (`withRateLimit`) demi mencegah serangan spamming/brute force API.
3.  **Correlation ID**: Middleware `withCorrelationID` memvalidasi atau membuat ID korelasi terdistribusi unik untuk audit log.
4.  **Startup Fail-Fast**: Fungsi `Ping()` dijalankan saat booting ke PostgreSQL dan Redis. Jika salah satu offline, aplikasi akan langsung melakukan `log.Fatalf` (crash) agar terdeteksi oleh orkestrator kontainer (mencegah mode dummy).

---

## 🔄 Integrasi Kafka (Payload Ter-enrich)
Data payload event `order.created` tidak hanya mengirim ID pesanan, melainkan **diperkaya (enriched)** dengan data berat, dimensi (L, W, H), total tarif, jarak tempuh, kota pengirim, dan kota penerima. Ini berguna agar ETL Service dan Notification Service tidak perlu memanggil ulang (query balik) database Order.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Buka tab "Create Order"** pada Dashboard UI.
2.  Masukkan data paket (berat, panjang, lebar, tinggi), isi kota asal (Bandung) dan tujuan (Jakarta).
3.  Klik **Submit** dan tunjukkan AWB (Nomor Resi) berhasil terbit lengkap dengan tarif yang dihitung otomatis.
4.  *Penjelasan Teknis*: Request dikirim melalui API Gateway port `8085` dengan path `/api/proxy/orders`. Gateway meneruskan ke microservice port `8082` sembari menyuntikkan API Key. Jarak dihitung menggunakan rumus Haversine, disimpan di database Postgres, dan event `order.created` dipublikasikan ke Kafka secara asinkron.
