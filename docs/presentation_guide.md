# 📢 Panduan Presentasi Mikroservis PAPITON Express

Dokumen ini berisi rangkuman penjelasan teknis, alur data, dan langkah demonstrasi (*live demo*) untuk **setiap service** dalam proyek Tugas Besar Cloud Computing ini. Gunakan dokumen ini sebagai contekan (*cheat sheet*) saat presentasi di depan dosen/penguji.

---

## 🗺️ Gambaran Umum Aliran Jaringan (API Gateway / Proxy)
Sebelum menjelaskan tiap service, jelaskan bahwa seluruh request frontend/client masuk melalui **ETL Proxy / API Gateway (Port 8085)**, yang secara otomatis:
1.  Menyuntikkan **`X-API-Key`** (`dev-papiton-api-key-2024`) untuk autentikasi internal.
2.  Menyuntikkan/meneruskan **`X-Correlation-ID`** untuk pelacakan transaksi terdistribusi.

---

## 📦 1. Order & Tariff Service (Layanan Pemesanan & Tarif)
Layanan Go-based transaksional untuk memproses pemesanan baru dan menghitung ongkos kirim.

### A. Detail Teknis & Infrastruktur
*   **Port Internal**: `8080` (Container) ➔ **Port Host**: `8082`
*   **Database**: PostgreSQL (`papiton_order_tariff_service_db`), tabel `orders`.
*   **Caching**: Redis (`order-redis`) untuk menyimpan rute jarak koordinat kota (TTL 24 jam) guna menghemat daya komputasi.
*   **Kafka Event**: Mempublikasikan event `order.created` ke topik `papiton.events.order` dengan *enriched payload* (memuat berat paket, dimensi, kota pengirim/penerima, tarif total, dan jarak).

### B. Fitur Keandalan (Resiliency)
*   **Rate Limiter**: Batas 100 RPM per IP (`withRateLimit`).
*   **Startup Fail-Fast**: Aplikasi langsung crash saat booting jika koneksi Postgres atau Redis terputus (`db.Ping()`), menghindari transaksi berjalan dalam kondisi lumpuh.

### C. Cara Live Demo:
1.  Buka tab **Create Order** di Dashboard UI.
2.  Masukkan data pengiriman (Bandung ke Jakarta), isi dimensi, lalu klik *Submit*.
3.  Tunjukkan bahwa AWB berhasil digenerate dan tarif dihitung otomatis.
4.  *(Di Balik Layar)*: Redis menyimpan cache jarak kota, dan event `order.created` dikirim ke broker Kafka.

---

## 🚚 2. Shipping & Dispatch Service (Layanan Kurir & Penugasan)
Layanan Go-based untuk pencarian kurir terdekat secara otomatis (*Auto-Dispatch*) dan pelacakan GPS.

### A. Detail Teknis & Infrastruktur
*   **Port Internal**: `8080` (Container) ➔ **Port Host**: `8081`
*   **Database**: 
    *   PostgreSQL (`shipping_test_db`) untuk tabel armada `couriers` dan transaksi penugasan `dispatches`.
    *   MongoDB untuk koordinat GPS kurir real-time.
*   **Kafka Event**: Mempublikasikan event `package.picked_up` / `dispatch.assigned` ke topik `papiton.events.shipping` dengan payload yang diperkaya jenis kendaraan kurir (`vehicle_type`).

### B. Fitur Keandalan (Resiliency)
*   **Decoupled Tracing**: Menangkap header `X-Correlation-ID` untuk memantau performa *matching* kurir terdekat.
*   **Startup Fail-Fast**: Crash jika Postgres atau MongoDB GPS offline saat inisialisasi.

### C. Cara Live Demo:
1.  Setelah order dibuat, masuk ke tab **Dispatcher UI** di Dashboard.
2.  Masukkan ID AWB dan klik **Auto-Dispatch**.
3.  Sistem secara otomatis mencari kurir terdekat yang berstatus `AVAILABLE` di wilayah asal, mengubah statusnya menjadi `ON_DUTY`, dan menyimpannya di DB.
4.  Tunjukkan status AWB kini berubah menjadi `ASSIGNED` dan kurir siap bergerak.

---

## 🏢 3. Warehouse & Inventory Service (Layanan Gudang & Manifest)
Layanan Go-based transaksional untuk operasional hub logistik, penyusunan barang masuk (*inbound*), dan truk manifest keberangkatan.

### A. Detail Teknis & Infrastruktur
*   **Port Internal**: `8080` (Container) ➔ **Port Host**: `8080`
*   **Database**: PostgreSQL (`papiton_warehouse`), tabel `warehouses`, `inbound_packages`, `manifests`, `manifest_packages`, dan `sorting_lanes`.
*   **Kafka Event**: Mempublikasikan event `package.in_transit` ke topik `papiton.events.tracking` setiap kali paket dipindai masuk ke dalam hub transit.

### B. Fitur Keandalan (Resiliency)
*   Mengimplementasikan middleware keamanan `requireAPIKey` untuk menolak request pemindaian yang tidak sah dari luar jaringan logistik.

### C. Cara Live Demo:
1.  Masuk ke menu **Warehouse Scan** di Dashboard Portal.
2.  Masukkan nomor AWB aktif, pilih ID Gudang transit (contoh: `WH-JKT`), lalu klik **Proses Scan Inbound**.
3.  Sistem akan memperbarui status logistik paket di gudang terkait dan memicu pengiriman event asinkron ke Kafka.

---

## 🔍 4. Tracking & Log Event Service (Layanan Pelacakan Resi)
Layanan Go-based NoSQL dengan throughput penulisan tinggi untuk merekam dan menampilkan riwayat perjalanan paket.

### A. Detail Teknis & Infrastruktur
*   **Port Internal**: `8080` (Container) ➔ **Port Host**: `8083`
*   **Database**: MongoDB (`tracking_db`), collection `tracking_logs` (di-index pada `{resi_id: 1, timestamp: -1}`).
*   **Kafka Listener**: Secara aktif mengonsumsi data dari seluruh topik Kafka (`order`, `shipping`, `tracking`) lalu merapikannya secara sekuensial ke dalam satu collection history.

### B. Fitur Keandalan (Resiliency)
*   **Idempotency Check**: Mengecek apakah `event_id` sudah pernah diproses sebelumnya di MongoDB untuk menghindari duplikasi penulisan log perjalanan akibat *network retry* di Kafka.

### C. Cara Live Demo:
1.  Masuk ke tab **Cek Resi / Tracking** di Dashboard.
2.  Masukkan nomor AWB pengiriman, lalu klik *Track*.
3.  Tunjukkan riwayat log logistik yang berurutan mulai dari `CREATED` (Order Svc) ➔ `ASSIGNED` (Shipping Svc) ➔ `AT_HUB` (Warehouse Svc). 
4.  Data ini ditarik secara dinamis dari database NoSQL MongoDB.

---

## 🔔 5. Notification & Messaging Service (Layanan Notifikasi)
Layanan Go-based asinkron (worker) yang bertugas mengirim email SMTP dan push notification FCM ke perangkat ponsel client.

### A. Detail Teknis & Infrastruktur
*   **Port Internal**: `8080` (Container) ➔ **Port Host**: `8084`
*   **Penyimpanan**: PostgreSQL (`notification_db`), tabel `notification_logs` untuk audit trail.
*   **Kafka Listener**: Berlangganan secara asinkron ke seluruh topik logistik untuk mendeteksi perubahan status paket yang memerlukan interaksi dengan pengguna.

### B. Fitur Keandalan (Resiliency)
*   **Exponential Backoff Retry**: Jika koneksi internet ke mail server SMTP atau FCM Firebase putus, pengiriman akan dicoba kembali sebanyak 5 kali (`1s`, `2s`, `4s`, `8s`, `16s`).
*   **Dead Letter Queue (DLQ)**: Jika tetap gagal setelah 5 kali, event dipindahkan ke topik `papiton.dlq.events` agar tidak memblokir antrean pesan lainnya (*non-blocking*).

### C. Cara Live Demo:
1.  Tunjukkan log konsol kontainer `notification-app` saat Anda membuat order atau melakukan scan inbound gudang.
2.  Layanan ini akan otomatis memicu pengiriman email konfirmasi / simulasi push FCM ke pelanggan.
3.  Riwayat sukses/gagal dicatat di tabel `notification_logs` di database `notification_db`.

---

## 📊 6. ETL Pipeline & Data Warehouse (Analisis ML & OLAP)
Sistem Python-based (ETL Consumer & Dashboard Server) yang memproses data mentah transaksional menjadi data analitis, memprediksi ETA menggunakan Machine Learning, dan melayani Visualisasi Dashboard.

### A. Detail Teknis & Infrastruktur
*   **Port Host**: `8085` (Gateway & API Metrics Server)
*   **Database**: PostgreSQL Data Warehouse (`papiton_dwh`) dengan skema Star Schema (`dim_date`, `dim_location`, `dim_warehouse`, `dim_service`, `fact_shipment`, `fact_notification`).
*   **Machine Learning**: Regresi Linear (Scikit-Learn) menggunakan file model serialisasi `/app/eta_model.pkl`.
*   **Decoupled Architecture**: Membaca data langsung dari enriched event Kafka. Database operasional (`ORDER_ENGINE`, dll.) hanya diakses sebagai engine fallback opsional.

### B. Fitur Keandalan (Resiliency)
*   **WaitGroup pada Kafka Consumer**: Menggunakan `sync.WaitGroup` agar background threads membersihkan koneksi dengan aman saat shutdown (*graceful shutdown*).
*   **Locks ML**: Menggunakan `threading.Lock()` pada file `eta_model.pkl` saat training/prediksi berjalan bersamaan untuk mencegah race condition/kerusakan file model.

### C. Cara Live Demo:
1.  Tunjukkan halaman **Dashboard Analitis utama** di `index.html`.
2.  Buktikan bahwa diagram pendapatan bulanan, volume gudang, dan rating driver ter-update otomatis sesaat setelah event inbound/dispatch dijalankan di tab portal.
3.  Jelaskan bahwa sistem secara otomatis **melatih ulang (retraining) model regresi linear** secara asinkron setiap kali status paket berubah menjadi **`DELIVERED`** untuk terus mengoptimalkan prediksi ETA berikutnya.
