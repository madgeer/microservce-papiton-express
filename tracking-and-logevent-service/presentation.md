# 🔍 Tracking & Log Event Service — Panduan Presentasi
**Direktori: `/tracking-and-logevent-service`**

Layanan Go NoSQL berkinerja tinggi untuk mencatat audit log pelacakan paket dan menampilkan riwayat resi secara sekuensial.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Go (Golang) menggunakan `net/http` standard library.
*   **Database**: MongoDB (`tracking_db` - collection `tracking_logs`). Dipilih NoSQL MongoDB karena performa penulisan log perjalanan paket logistik berfrekuensi sangat tinggi (*high-write throughput*).
*   **Kafka Listener**: Konsumer Kafka yang mendengarkan 3 topik sekaligus (`papiton.events.order`, `papiton.events.shipping`, `papiton.events.tracking`).

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Otentikasi API Key**: Middleware `requireAPIKey` memproteksi API pembacaan log publik pada port `8083`.
2.  **Rate Limiter**: Dilindungi oleh middleware `withRateLimit` (100 RPM per IP).
3.  **Idempotensi (Event Idempotency)**: Dikarenakan Kafka dapat mengirim ulang pesan yang sama jika terjadi ketidakstabilan jaringan (*at-least-once delivery*), Tracking Service memeriksa apakah `event_id` pesan sudah pernah dicatat di MongoDB. Jika ya, pesan dibuang (*silently discarded*) untuk mencegah duplikasi histori paket.
4.  **Startup Fail-Fast**: Crash saat booting (`log.Fatalf`) jika MongoDB offline saat dipanggil oleh `client.Ping()`.

---

## 🔄 Integrasi Kafka (Event Consumption)
Mengkonsumsi event asinkron dari topik `papiton.events.order` (tipe `order.created`), `papiton.events.shipping` (tipe `package.picked_up`), dan `papiton.events.tracking` (tipe `package.in_transit`), menyatukan datanya, lalu menyimpannya secara sekuensial di MongoDB.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Buka tab "Cek Resi / Tracking"** pada Dashboard UI.
2.  Masukkan nomor AWB aktif yang telah melewati pembuatan order, auto-dispatch, dan scan inbound.
3.  Klik **Track** dan tunjukkan log history perjalanan yang tersusun rapi secara kronologis terbalik (terbaru di atas) lengkap dengan lokasi hub dan aktivitasnya.
4.  *Penjelasan Teknis*: Request melewati API Gateway `/api/proxy/tracking?resi_id=...` ➔ diteruskan ke microservice port `8083`. Sistem mencari dokumen AWB terkait di MongoDB, mengurutkannya berdasarkan timestamp desc, dan mengembalikan data dalam format JSON.
