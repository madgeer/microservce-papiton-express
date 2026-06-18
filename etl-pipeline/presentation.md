# 📊 ETL Pipeline & Data Warehouse (OLAP) — Panduan Presentasi
**Direktori: `/etl-pipeline`**

Layanan Python-based untuk sinkronisasi Data Warehouse (DWH), Server API Gateway/Proxy, dan Machine Learning ETA Prediction.

---

## 💻 Aspek Teknis (Tech Stack & DB)
*   **Bahasa Pemrograman**: Python 3.10+ (menggunakan library `pandas`, `sqlalchemy`, `scikit-learn`, `pymongo`, `kafka-python`).
*   **Database**: PostgreSQL Data Warehouse (`papiton_dwh`) dengan skema Star Schema (`dim_date`, `dim_location`, `dim_warehouse`, `dim_service`, `fact_shipment`, `fact_notification`).
*   **Machine Learning**: Linear Regression model (`eta_model.pkl`) untuk estimasi waktu pengiriman secara dinamis.
*   **API Gateway / Proxy**: Port `8085` (`DashboardAPIHandler`) bertindak sebagai gateway yang mem-proxy panggilan frontend dashboard ke microservices backend Go.

---

## 🔒 Fitur Keandalan & Keamanan (Materi Uji Dosen)
1.  **Header Injection**: Gateway Proxy pada port `8085` menyuntikkan header `X-API-Key` dan `X-Correlation-ID` ke setiap request yang diarahkan ke internal microservices backend Go agar terautentikasi dan terlacak dengan aman.
2.  **Locks ML Model File**: Menggunakan `threading.Lock()` (`model_lock`) saat training atau melakukan prediksi agar thread paralel tidak mengalami race condition atau merusak file biner model `eta_model.pkl`.
3.  **WaitGroup pada Kafka Consumer**: Menggunakan `sync.WaitGroup` agar background threads membersihkan koneksi secara aman ketika container dimatikan (*graceful shutdown*).
4.  **Decoupled Architecture**: Membaca data langsung dari payload Kafka yang diperkaya (*enriched Kafka payload*). Engine database transaksional operasional (`ORDER_ENGINE`, dll.) hanya diakses sebagai engine fallback opsional.

---

## 🔄 Integrasi Kafka & Loop Machine Learning
*   Mengkonsumsi event logistik untuk mengisi tabel dimensi dan fakta di Data Warehouse (`papiton_dwh`).
*   **ML Retraining Loop**: Setiap kali event berstatus **`DELIVERED`** diterima, ETL Service menghitung selisih durasi aktual dan prediksi (*ETA prediction error*), lalu secara asinkron memicu *background thread* untuk melatih ulang model regresi linear menggunakan seluruh riwayat pengiriman sukses di DWH (minimal 3 record historis). Model baru lalu disimpan ke `eta_model.pkl`.

---

## 🎭 Skenario Demonstrasi (Live Demo)
1.  **Tunjukkan grafik visualisasi** pada halaman dashboard utama (`index.html`).
2.  Lakukan transaksi baru di portal (order, dispatch, scan), lalu segarkan (*refresh*) dashboard.
3.  Tunjukkan bahwa total pendapatan, volume pengiriman bulanan, dan rating driver terupdate secara real-time.
4.  *Penjelasan Teknis*: Frontend melakukan fetch ke `/api/metrics` di port `8085`. Server Python mengeksekusi query SQL agregasi analitis ke Data Warehouse PostgreSQL (`papiton_dwh`) dan mengembalikan data metrik ter-agregasi dalam format JSON.
