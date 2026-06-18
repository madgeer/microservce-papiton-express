# 🚚 Shipping & Dispatch Service — Panduan Presentasi
**Direktori: `/shipping-service`**

Layanan Go transaksional untuk manajemen kurir, otomatisasi penugasan penjemputan (*Auto-Dispatch*), dan GPS tracking.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Go (Golang) menggunakan `net/http` standard library.
*   **Database**: 
    *   PostgreSQL (`shipping_test_db`) untuk tabel `couriers` (data armada) dan `dispatches` (tugas penugasan).
    *   MongoDB (`shipping_mongo`) untuk spasial logs koordinat GPS kurir (`locations` collection).
*   **Kafka Event**: Publisher untuk mempublikasikan event `package.picked_up` ke topik `papiton.events.shipping` ketika status penugasan berubah.

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Otentikasi API Key**: Middleware `requireAPIKey` memvalidasi header `X-API-Key` pada port `8081` agar database internal aman dari manipulasi.
2.  **Rate Limiter**: Dilindungi oleh middleware `withRateLimit` (100 RPM per IP).
3.  **Correlation ID**: Meneruskan header `X-Correlation-ID` untuk memantau performa *matching* kurir terdekat.
4.  **Startup Fail-Fast**: Mengecek koneksi PostgreSQL via `db.Ping()` dan MongoDB via `client.Ping()` saat booting. Jika salah satu database tidak sehat, aplikasi langsung dihentikan (`log.Fatalf`).

---

## 🔄 Integrasi Kafka (Payload Ter-enrich)
Event `package.picked_up` / `dispatch.assigned` yang dipublikasikan diperkaya (*enriched*) dengan informasi `vehicle_type` (tipe kendaraan kurir: Motor/Van/Truk) serta `route_instruction` dari algoritma rute. Ini bertujuan agar ETL Service dapat menghitung pendapatan dan rating kurir langsung dari payload tanpa menyentuh database Shipping.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Buka tab "Courier Dispatch"** pada Dashboard UI.
2.  Masukkan nomor AWB aktif yang baru dibuat, pilih zona penjemputan (Bandung), lalu klik **Auto-Dispatch**.
3.  Tunjukkan kurir terdekat berstatus `AVAILABLE` (misal: C-001) terpilih, status kurir berubah menjadi `ON_DUTY` di database PostgreSQL, dan instruksi rute dibuat.
4.  *Penjelasan Teknis*: Request melewati API Gateway `/api/proxy/dispatch` ➔ diteruskan ke microservice port `8081`. Sistem mengambil daftar kurir di zona terkait, mencocokkan kurir tersedia, mencatat transaksi penugasan, dan mengirimkan pesan asinkron `dispatch.assigned` ke Kafka.
