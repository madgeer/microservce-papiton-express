# 🔔 Notification & Messaging Service — Panduan Presentasi
**Direktori: `/notification-and-messaging-service`**

Layanan Go asinkron (worker) untuk mengirimkan notifikasi Email SMTP dan Push Notification FCM Firebase secara real-time.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Go (Golang) menggunakan `net/http` standard library.
*   **Database**: PostgreSQL (`notification_db` pada container `notification-db:5436`), tabel `notification_logs` untuk menyimpan audit logs pengiriman notifikasi demi mencegah spamming.
*   **Kafka Listener**: Konsumer asinkron untuk mendengarkan topik order, shipping, dan tracking.

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Exponential Backoff Retry**: Jika pengiriman email atau push notification gagal akibat kendala jaringan eksternal ke SMTP Google atau FCM Firebase, sistem akan mencoba mengirim ulang sebanyak 5 kali dengan jeda waktu yang meluas (`1s`, `2s`, `4s`, `8s`, `16s`).
2.  **Dead Letter Queue (DLQ)**: Jika setelah 5x pengiriman tetap gagal, pesan dipindahkan ke topik antrean khusus `papiton.dlq.events` agar tidak menyumbat antrean pesan baru (*non-blocking queue*).
3.  **Startup Fail-Fast**: Crash jika PostgreSQL `notification_db` mati saat inisialisasi booting.

---

## 🔄 Integrasi Kafka (Event Consumption)
Mendengar event asinkron dari topik Kafka untuk:
*   `order.created` ➔ Mengirimkan Email Konfirmasi Pembayaran dan Rincian Paket.
*   `package.picked_up` ➔ Mengirimkan Push Notifikasi kurir dalam perjalanan menjemput paket.
*   `package.in_transit` ➔ Mengirimkan Push Notifikasi paket sedang transit di hub gudang.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Lihat log output** dari terminal kontainer `notification-app` saat Anda membuat order atau melakukan scan inbound di portal.
2.  Tunjukkan bahwa server asinkron langsung mendeteksi event dan mengeksekusi pengiriman notifikasi secara paralel.
3.  *Penjelasan Teknis*: Notification Service bertindak sebagai worker murni yang berjalan di background. Begitu Kafka menyalurkan event, layanan ini membaca data, memilah jenis notifikasi, mengeksekusi pengiriman via protokol SMTP / REST FCM, dan mencatat statusnya ke database PostgreSQL.
