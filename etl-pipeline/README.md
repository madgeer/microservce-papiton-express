# 📊 Panduan Uji Coba Data Engineering & ML Pipeline (etl-pipeline)

Layanan **etl-pipeline** (`etl-service`) berbasis Python ini bertugas mengonsumsi data enriched event dari Apache Kafka, membersihkannya, menyimpannya ke **PostgreSQL Data Warehouse (DWH)**, serta melatih model **Machine Learning (Linear Regression)** secara otomatis untuk memprediksi durasi pengiriman (ETA).

Berikut adalah panduan lengkap cara mencoba dan memverifikasi fitur Data Engineering & Machine Learning secara langsung di sistem lokal Anda.

---

## 🛠️ 1. Prasyarat Sebelum Uji Coba
Pastikan seluruh 18 kontainer Docker menyala dengan sehat di root direktori proyek Anda:
```bash
docker compose up -d
```
Pastikan port API Gateway DWH (`8085`) dan database DWH (`5435`) dapat diakses.

---

## 🚀 2. Langkah-Langkah Live Demo (Alur Data & Retraining ML)

Ikuti skenario ini untuk melihat bagaimana data mengalir dari frontend ➔ Kafka ➔ ETL Pipeline ➔ Data Warehouse ➔ Retraining Model AI secara asinkron.

### Langkah 1: Pantau Log ETL Service secara Real-Time
Buka terminal baru di komputer Anda, lalu jalankan perintah berikut untuk mengamati aktivitas pipeline ETL:
```bash
docker compose logs -f etl-service
```

### Langkah 2: Buat Order Baru & Lacak AWB
1. Buka dashboard publik di browser Anda (`http://localhost/` atau buka `index.html` jika dilayani lokal).
2. Buat pesanan baru di tab **Buat Order** (misal dari kota Bandung ke Jakarta).
3. Catat nomor **AWB (Resi)** yang berhasil dibuat (misalnya: `BDG260619...`).
4. *Di Log ETL*: Anda akan melihat log masuk ketika event `order.created` ditangkap, menghitung berat volumetrik, memprediksi durasi ETA menggunakan ML, dan meng-INSERT data tersebut ke tabel `fact_shipment` di database DWH.

### Langkah 3: Lakukan Inbound Scan & Penugasan Kurir
1. Buka **Staff Portal** (`portal.html`), masuk ke tab **Warehouse Scan**.
2. Masukkan nomor AWB di form **Inbound Scan**, pilih gudang `WH-UPI`, lalu klik **Proses Inbound**.
3. Di tab **Dispatcher**, masukkan nomor AWB dan klik **Auto-Dispatch** untuk menugaskan kurir terdekat.
4. *Di Log ETL*: ETL mendeteksi pergerakan status paket dan langsung memperbarui kolom status serta performa pendapatan kurir di DWH.

### Langkah 4: Simulasikan Status DELIVERED (Retraining AI)
1. Di bagian bawah halaman Staff Portal, cari bagian **Simulasi Scan Barcode Manual**.
2. Masukkan nomor AWB Anda.
3. Pilih Kode Status Event: **`DELIVERED (Paket berhasil diterima oleh customer)`**.
4. Klik **Kirim Log Scan Manual**.
5. *Di Log ETL (Perhatikan Terminal Anda)*:
   * ETL menangkap event status `DELIVERED`.
   * ETL menghitung durasi waktu pengiriman aktual dalam jam (`actual_duration_hours`).
   * ETL mencatat tingkat kesalahan prediksi model (`eta_prediction_error`).
   * **PENTING**: ETL akan langsung meluncurkan thread latar belakang untuk melakukan **Retraining Ulang Model Machine Learning (Linear Regression)** menggunakan data historis baru! Anda akan melihat log seperti:
     ```text
     Triggering machine learning model training loop...
     Machine learning model successfully trained on X records and saved to /app/eta_model.pkl!
     ```

---

## 🔍 3. Verifikasi Data Langsung di Database DWH

Untuk memastikan data benar-benar tersimpan di database Data Warehouse (`papiton_dwh`), Anda bisa masuk ke container database DWH dan menjalankan query analisis (OLAP).

### A. Masuk ke PostgreSQL DWH
Jalankan perintah berikut di terminal Anda untuk masuk ke antarmuka PostgreSQL client di dalam container DWH:
```bash
docker exec -it dwh-db psql -U postgres -d papiton_dwh
```

### B. Kueri SQL Analisis untuk Verifikasi
Setelah masuk ke prompt psql (`papiton_dwh=#`), jalankan kueri-kueri berguna berikut untuk mengaudit data:

1. **Melihat metrik pengiriman dan akurasi prediksi ML:**
   ```sql
   SELECT 
       awb, 
       order_status, 
       tarif_total,
       distance_km, 
       package_weight,
       predicted_duration_hours AS prediksi_ml_jam,
       actual_duration_hours AS aktual_pengiriman_jam,
       eta_prediction_error AS selisih_error_jam
   FROM fact_shipment;
   ```

2. **Melihat performa notifikasi pelanggan (tabel sub-fakta):**
   ```sql
   SELECT 
       fn.awb, 
       dn.channel, 
       dn.event_type, 
       fn.success, 
       fn.notif_at 
   FROM fact_notification fn
   JOIN dim_notification_type dn ON fn.notif_type_key = dn.notif_type_key;
   ```

3. **Melihat data agregat (Summary metrics untuk dashboard):**
   ```sql
   SELECT 
       COUNT(*) AS total_pengiriman,
       ROUND(AVG(actual_duration_hours), 2) AS rata_rata_durasi_jam,
       ROUND(AVG(ABS(eta_prediction_error)), 2) AS rata_rata_kesalahan_prediksi_jam
   FROM fact_shipment
   WHERE order_status = 'DELIVERED';
   ```

Ketik `\q` lalu tekan `Enter` untuk keluar dari PostgreSQL prompt.

---

## 🧪 4. Menjalankan Uji Coba Otomatis (Skrip Python)
Jika Anda malas mengetik secara manual di antarmuka web, Anda bisa menggunakan skrip pengujian otomatis yang menyimulasikan seluruh 6 alur di atas dalam 1 kali eksekusi:

Jalankan perintah berikut di terminal komputer lokal Anda:
```bash
python -c "import urllib.request; print('Python ready')"
```
Lalu jalankan skrip pengujian:
```bash
python ../C:/Users/HP/.gemini/antigravity-cli/brain/cc4cb507-14a3-489c-82d6-9c95042bcd81/scratch/test_flow.py
```
*(Skrip tersebut akan otomatis membuat order, menugaskan kurir, memproses inbound scan gudang, menyusun manifest, memasukkan paket ke manifest, dan merilis status keberangkatan manifest tanpa ada error)*.
