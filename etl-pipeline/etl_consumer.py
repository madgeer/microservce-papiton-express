import os
import json
import datetime
import time
import pickle
import threading
from concurrent.futures import ThreadPoolExecutor
import numpy as np
import pandas as pd
from sqlalchemy import create_engine, text
from pymongo import MongoClient
from kafka import KafkaConsumer
from sklearn.linear_model import LinearRegression
from http.server import BaseHTTPRequestHandler, HTTPServer

# --- CONFIGURATION FROM ENV OR DEFAULTS ---
KAFKA_BROKER = os.getenv("KAFKA_BROKER", "kafka:9092")
ORDER_DB_URL = os.getenv("ORDER_DB_URL", "postgresql://postgres:admin123@order-db:5432/papiton_order_tariff_service_db")
SHIPPING_DB_URL = os.getenv("SHIPPING_DB_URL", "postgresql://user:password@shipping-db:5432/shipping_test_db")
WAREHOUSE_DB_URL = os.getenv("WAREHOUSE_DB_URL", "postgresql://postgres:postgres@warehouse-db:5432/papiton_warehouse")
DWH_DB_URL = os.getenv("DWH_DB_URL", "postgresql://postgres:dwhpassword@dwh-db:5432/papiton_dwh")
MONGO_URI = os.getenv("MONGO_URI", "mongodb://tracking-mongo:27017")
MONGO_DB_NAME = os.getenv("MONGO_DB_NAME", "tracking_db")

# URL internal service — dikonfigurasi via env var, tidak hardcode nama container
ORDER_SERVICE_URL    = os.getenv("ORDER_SERVICE_URL",    "http://order-app:8080")
SHIPPING_SERVICE_URL = os.getenv("SHIPPING_SERVICE_URL", "http://shipping-app:8080")
WAREHOUSE_SERVICE_URL= os.getenv("WAREHOUSE_SERVICE_URL","http://warehouse-app:8080")
TRACKING_SERVICE_URL = os.getenv("TRACKING_SERVICE_URL", "http://tracking-app:8080")

print("Initializing database engines...")
ORDER_ENGINE = create_engine(ORDER_DB_URL)
SHIPPING_ENGINE = create_engine(SHIPPING_DB_URL)
WAREHOUSE_ENGINE = create_engine(WAREHOUSE_DB_URL)
DWH_ENGINE = create_engine(DWH_DB_URL)

MODEL_PATH = "/app/eta_model.pkl"
# Lock untuk mencegah race condition saat training dan prediksi model secara bersamaan
model_lock = threading.Lock()

def train_model():
    print("Triggering machine learning model training loop...")
    with model_lock:
        try:
            # Read historical delivered data from DWH
            with DWH_ENGINE.connect() as conn:
                query = """
                    SELECT f.distance_km, f.package_weight, f.volumetric_weight,
                           (CASE WHEN s.service_type = 'EXPRESS' THEN 1 ELSE 0 END) as is_express,
                           f.actual_duration_hours
                    FROM fact_shipment f
                    JOIN dim_service s ON f.service_key = s.service_key
                    WHERE f.order_status = 'DELIVERED' AND f.actual_duration_hours IS NOT NULL
                """
                df = pd.read_sql(query, conn)

            if len(df) < 3:
                print(f"Not enough training data in DWH (have {len(df)} records, need at least 3). Skipping training.")
                return

            X = df[['distance_km', 'package_weight', 'volumetric_weight', 'is_express']].values
            y = df['actual_duration_hours'].values

            model = LinearRegression()
            model.fit(X, y)

            # Save model to file
            with open(MODEL_PATH, 'wb') as f:
                pickle.dump(model, f)
            print(f"Machine learning model successfully trained on {len(df)} records and saved to {MODEL_PATH}!")
        except Exception as e:
            print(f"Error training machine learning model: {e}")

def predict_duration(distance, weight, vol_weight, is_express):
    # Rule-based heuristic fallback (e.g. 2 hrs per 50km for express, 5 hrs per 50km for regular + weight factor)
    base_duration = (distance / 50.0) * (2.0 if is_express else 5.0) + (weight * 0.5)
    if base_duration < 1.0:
        base_duration = 1.0
        
    if not os.path.exists(MODEL_PATH):
        return round(base_duration, 2)
        
    try:
        with model_lock:
            with open(MODEL_PATH, 'rb') as f:
                model = pickle.load(f)
        features = np.array([[distance, weight, vol_weight, 1 if is_express else 0]])
        pred = model.predict(features)[0]
        if pred < 0.5:
            pred = 0.5
        return round(float(pred), 2)
    except Exception as e:
        print(f"Error in ML prediction: {e}")
        return round(base_duration, 2)
# --- 1. DWH DATABASE INITIALIZATION ---
def init_dwh_tables():
    print("Checking and creating DWH tables...")
    with DWH_ENGINE.begin() as conn:
        # 1. Dimensions
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_date (
                date_key INT PRIMARY KEY,
                full_date DATE NOT NULL,
                day_of_week VARCHAR(15) NOT NULL,
                month_num SMALLINT NOT NULL,
                month_name VARCHAR(15) NOT NULL,
                quarter SMALLINT NOT NULL,
                year SMALLINT NOT NULL,
                is_weekend BOOLEAN NOT NULL
            );
        """))
        
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_location (
                location_key SERIAL PRIMARY KEY,
                province VARCHAR(50) NOT NULL,
                city_kabupaten VARCHAR(50) NOT NULL,
                district_kecamatan VARCHAR(50) NOT NULL,
                subdistrict_kelurahan VARCHAR(50) NOT NULL,
                UNIQUE (province, city_kabupaten, district_kecamatan, subdistrict_kelurahan)
            );
        """))

        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_warehouse (
                warehouse_key SERIAL PRIMARY KEY,
                warehouse_id VARCHAR(50) UNIQUE NOT NULL,
                warehouse_name VARCHAR(100) NOT NULL,
                city VARCHAR(50) NOT NULL,
                region VARCHAR(50) NOT NULL,
                warehouse_type VARCHAR(20) NOT NULL
            );
        """))

        # Pre-populate dim_warehouse with some defaults if empty
        conn.execute(text("""
            INSERT INTO dim_warehouse (warehouse_id, warehouse_name, city, region, warehouse_type)
            VALUES 
                ('WH-BDG', 'Hub Utama Bandung', 'Bandung', 'Jawa Barat', 'HUB'),
                ('WH-JKT', 'Hub Regional Jakarta', 'Jakarta', 'DKI Jakarta', 'HUB'),
                ('WH-SUB', 'Hub Timur Surabaya', 'Surabaya', 'Jawa Timur', 'HUB'),
                ('WH-UPI', 'Hub Transit UPI', 'Bandung', 'Jawa Barat', 'TRANSIT')
            ON CONFLICT (warehouse_id) DO NOTHING;
        """))

        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_service (
                service_key SERIAL PRIMARY KEY,
                service_type VARCHAR(20) UNIQUE NOT NULL,
                has_insurance BOOLEAN DEFAULT FALSE,
                has_packing BOOLEAN DEFAULT FALSE,
                base_price_per_km NUMERIC,
                insurance_fee NUMERIC
            );
        """))

        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_notification_type (
                notif_type_key SERIAL PRIMARY KEY,
                channel VARCHAR(20) NOT NULL,
                event_type VARCHAR(50) NOT NULL,
                UNIQUE (channel, event_type)
            );
        """))
        
        # 2. Facts
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS fact_shipment (
                shipment_key SERIAL PRIMARY KEY,
                awb VARCHAR(50) UNIQUE NOT NULL,
                date_key INT NOT NULL REFERENCES dim_date(date_key),
                origin_location_key INT NOT NULL REFERENCES dim_location(location_key),
                destination_location_key INT NOT NULL REFERENCES dim_location(location_key),
                courier_id VARCHAR(50) NOT NULL DEFAULT 'N/A',
                warehouse_key INT NOT NULL REFERENCES dim_warehouse(warehouse_key),
                service_key INT NOT NULL REFERENCES dim_service(service_key),
                order_status VARCHAR(50) NOT NULL,
                tarif_total NUMERIC NOT NULL,
                distance_km NUMERIC NOT NULL,
                package_weight NUMERIC NOT NULL,
                volumetric_weight NUMERIC NOT NULL,
                predicted_duration_hours NUMERIC,
                actual_duration_hours NUMERIC,
                eta_prediction_error NUMERIC,
                driver_earnings NUMERIC DEFAULT 0.0,
                driver_rating NUMERIC DEFAULT 0.0,
                notification_count INT DEFAULT 0,
                etl_loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        """))

        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS fact_notification (
                notif_key SERIAL PRIMARY KEY,
                awb VARCHAR(50) NOT NULL,
                date_key INT NOT NULL REFERENCES dim_date(date_key),
                notif_type_key INT NOT NULL REFERENCES dim_notification_type(notif_type_key),
                success BOOLEAN NOT NULL,
                notif_at TIMESTAMP NOT NULL,
                etl_loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        """))

        # Migrations to handle cases where tables were already created with old schema
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS predicted_duration_hours NUMERIC;"))
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS actual_duration_hours NUMERIC;"))
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS eta_prediction_error NUMERIC;"))
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS courier_id VARCHAR(50) NOT NULL DEFAULT 'N/A';"))
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS driver_earnings NUMERIC DEFAULT 0.0;"))
        conn.execute(text("ALTER TABLE fact_shipment ADD COLUMN IF NOT EXISTS driver_rating NUMERIC DEFAULT 0.0;"))
        conn.execute(text("ALTER TABLE fact_shipment DROP COLUMN IF EXISTS courier_profile_key;"))
        conn.execute(text("DROP TABLE IF EXISTS dim_courier_profile;"))

        # Pre-populate dim_date from 2025 to 2030 if empty
        date_count = conn.execute(text("SELECT COUNT(*) FROM dim_date")).scalar()
        if date_count == 0:
            print("Populating dim_date dimension tables...")
            conn.execute(text("""
                INSERT INTO dim_date (date_key, full_date, day_of_week, month_num, month_name, quarter, year, is_weekend)
                SELECT 
                    TO_CHAR(d, 'YYYYMMDD')::INT AS date_key,
                    d::DATE AS full_date,
                    TO_CHAR(d, 'Day') AS day_of_week,
                    EXTRACT(MONTH FROM d)::SMALLINT AS month_num,
                    TO_CHAR(d, 'Month') AS month_name,
                    EXTRACT(QUARTER FROM d)::SMALLINT AS quarter,
                    EXTRACT(YEAR FROM d)::SMALLINT AS year,
                    CASE WHEN EXTRACT(ISODOW FROM d) IN (6, 7) THEN TRUE ELSE FALSE END AS is_weekend
                FROM generate_series('2025-01-01'::DATE, '2030-12-31'::DATE, '1 day'::interval) d;
            """))
            print("dim_date populated successfully!")

# --- 2. PIPELINE HELPER FUNCTIONS ---
def get_or_create_date_key(conn, timestamp):
    date_key = int(timestamp.strftime('%Y%m%d'))
    # Make sure it exists in dim_date
    exists = conn.execute(text("SELECT 1 FROM dim_date WHERE date_key = :date_key"), {"date_key": date_key}).scalar()
    if not exists:
        conn.execute(text("""
            INSERT INTO dim_date (date_key, full_date, day_of_week, month_num, month_name, quarter, year, is_weekend)
            VALUES (
                :date_key, :full_date, :day_of_week, :month_num, :month_name, :quarter, :year, :is_weekend
            )
        """), {
            "date_key": date_key,
            "full_date": timestamp.date(),
            "day_of_week": timestamp.strftime('%A'),
            "month_num": timestamp.month,
            "month_name": timestamp.strftime('%B'),
            "quarter": (timestamp.month - 1) // 3 + 1,
            "year": timestamp.year,
            "is_weekend": timestamp.weekday() >= 5
        })
    return date_key

def get_or_create_location_key(conn, city_name):
    if not city_name or pd.isnull(city_name):
        city_name = "Unknown"
    
    cleaned_city = str(city_name).strip().upper()
    # Simple mapping dictionary to normalize free text to DWH dimension standards (province, city/kabupaten, district, subdistrict)
    mapping = {
        "BANDUNG": {
            "province": "Jawa Barat",
            "city_kabupaten": "Kota Bandung",
            "district_kecamatan": "Coblong",
            "subdistrict_kelurahan": "Dago"
        },
        "JAKARTA": {
            "province": "DKI Jakarta",
            "city_kabupaten": "Kota Jakarta Selatan",
            "district_kecamatan": "Kebayoran Baru",
            "subdistrict_kelurahan": "Selong"
        },
        "SURABAYA": {
            "province": "Jawa Timur",
            "city_kabupaten": "Kota Surabaya",
            "district_kecamatan": "Gubeng",
            "subdistrict_kelurahan": "Gubeng"
        }
    }
    
    # Fallback if city name not in preset mapping
    loc_data = mapping.get(cleaned_city, {
        "province": "Jawa Barat" if "BDG" in cleaned_city or "BANDUNG" in cleaned_city else "Unknown",
        "city_kabupaten": f"Kota {city_name}" if "KOTA" not in cleaned_city.split() else city_name,
        "district_kecamatan": "Unknown",
        "subdistrict_kelurahan": "Unknown"
    })
    
    # Try inserting with conflict handling
    row = conn.execute(text("""
        INSERT INTO dim_location (province, city_kabupaten, district_kecamatan, subdistrict_kelurahan)
        VALUES (:province, :city_kabupaten, :district_kecamatan, :subdistrict_kelurahan)
        ON CONFLICT (province, city_kabupaten, district_kecamatan, subdistrict_kelurahan)
        DO UPDATE SET province = EXCLUDED.province
        RETURNING location_key;
    """), loc_data).fetchone()
    
    if row:
        return row[0]
    
    # Fallback query
    return conn.execute(text("""
        SELECT location_key FROM dim_location 
        WHERE province = :province AND city_kabupaten = :city_kabupaten 
          AND district_kecamatan = :district_kecamatan AND subdistrict_kelurahan = :subdistrict_kelurahan
    """), loc_data).scalar()

def parse_eta(eta_str, created_at):
    if not eta_str or pd.isnull(eta_str):
        return None
    try:
        # Try direct parsing
        return pd.to_datetime(eta_str)
    except Exception:
        # Failed to parse, check for string patterns like "2 Hours", "3 Days"
        try:
            cleaned = str(eta_str).lower().strip()
            parts = cleaned.split()
            if len(parts) >= 2:
                num = float(parts[0])
                unit = parts[1]
                if "hour" in unit:
                    return created_at + datetime.timedelta(hours=num)
                elif "day" in unit:
                    return created_at + datetime.timedelta(days=num)
                elif "min" in unit:
                    return created_at + datetime.timedelta(minutes=num)
        except Exception:
            pass
        # Fallback to created_at + 1 day
        return created_at + datetime.timedelta(days=1)

def handle_order_created(awb, event_metadata=None):
    """Proses order.created event menggunakan data dari Kafka payload (event_metadata).
    ETL tidak lagi query langsung ke order-db — data sudah dikirim oleh order-service."""
    print(f"Processing order.created event for AWB: {awb}")

    # Ekstrak data dari metadata event (sudah diperkaya oleh order-service)
    meta = event_metadata or {}
    service_type   = meta.get("service_type", "REGULAR")
    has_insurance  = bool(meta.get("has_insurance", False))
    has_packing    = bool(meta.get("has_packing", False))
    sender_city    = meta.get("sender_city", "Unknown")
    recipient_city = meta.get("recipient_city", "Unknown")
    package_weight = float(meta.get("package_weight", 0))
    package_length = float(meta.get("package_length", 0))
    package_width  = float(meta.get("package_width", 0))
    package_height = float(meta.get("package_height", 0))
    tarif_total    = float(meta.get("tarif_total", 0))
    distance_km    = float(meta.get("distance_km", 0))
    order_status   = meta.get("status", "Shipment Created")

    # Jika metadata kosong (event lama / backward-compat), fallback ke query order-db
    if not meta or distance_km == 0:
        print(f"[ETL Warning] Metadata kosong untuk AWB {awb}, fallback ke query order-db...")
        with ORDER_ENGINE.connect() as src_conn:
            order = src_conn.execute(text("SELECT * FROM orders WHERE awb = :awb"), {"awb": awb}).mappings().first()
        if not order:
            print(f"ERROR: Order AWB {awb} tidak ditemukan di DB. Event dilewati.")
            return
        service_type   = order["service_type"]
        has_insurance  = order["has_insurance"]
        has_packing    = order["has_packing"]
        sender_city    = order["sender_city"]
        recipient_city = order["recipient_city"]
        package_weight = float(order["package_weight"])
        package_length = float(order["package_length"])
        package_width  = float(order["package_width"])
        package_height = float(order["package_height"])
        tarif_total    = float(order["tarif_total"])
        distance_km    = float(order["distance"])
        order_status   = order["status"]

    # Warehouse default — akan diperbarui saat event inbound diterima
    warehouse_id = 'WH-BDG'

    with DWH_ENGINE.begin() as conn:
        # Load dim_service
        conn.execute(text("""
            INSERT INTO dim_service (service_type, has_insurance, has_packing, base_price_per_km, insurance_fee)
            VALUES (:service_type, :has_insurance, :has_packing, :base_price_per_km, :insurance_fee)
            ON CONFLICT (service_type) DO UPDATE SET
                has_insurance = EXCLUDED.has_insurance,
                has_packing = EXCLUDED.has_packing;
        """), {
            "service_type": service_type,
            "has_insurance": has_insurance,
            "has_packing": has_packing,
            "base_price_per_km": 1500 if service_type == 'EXPRESS' else 1000,
            "insurance_fee": 5000 if has_insurance else 0
        })

        created_at = datetime.datetime.now()
        date_key = get_or_create_date_key(conn, created_at)

        origin_location_key = get_or_create_location_key(conn, sender_city)
        destination_location_key = get_or_create_location_key(conn, recipient_city)

        warehouse_key = conn.execute(text("SELECT warehouse_key FROM dim_warehouse WHERE warehouse_id = :warehouse_id"), {"warehouse_id": warehouse_id}).scalar()
        service_key = conn.execute(text("SELECT service_key FROM dim_service WHERE service_type = :service_type"), {"service_type": service_type}).scalar()

        # Calculate Volumetric Weight
        volumetric_weight = (package_length * package_width * package_height) / 6000.0

        # ML Inference: Predict ETA duration in hours
        is_express = (service_type == 'EXPRESS')
        predicted_hours = predict_duration(
            distance=distance_km,
            weight=package_weight,
            vol_weight=volumetric_weight,
            is_express=is_express
        )

        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()

        params = {
            "awb": awb,
            "date_key": date_key,
            "origin_location_key": origin_location_key,
            "destination_location_key": destination_location_key,
            "courier_id": "N/A",
            "warehouse_key": warehouse_key,
            "service_key": service_key,
            "order_status": order_status,
            "tarif_total": tarif_total,
            "distance_km": distance_km,
            "package_weight": package_weight,
            "volumetric_weight": volumetric_weight,
            "predicted_duration_hours": predicted_hours,
            "driver_earnings": 0.0,
            "driver_rating": 0.0
        }

        if not exists:
            conn.execute(text("""
                INSERT INTO fact_shipment (
                    awb, date_key, origin_location_key, destination_location_key, courier_id,
                    warehouse_key, service_key, order_status, tarif_total, distance_km, package_weight,
                    volumetric_weight, predicted_duration_hours, driver_earnings, driver_rating
                ) VALUES (
                    :awb, :date_key, :origin_location_key, :destination_location_key, :courier_id,
                    :warehouse_key, :service_key, :order_status, :tarif_total, :distance_km, :package_weight,
                    :volumetric_weight, :predicted_duration_hours, :driver_earnings, :driver_rating
                )
            """), params)
            print(f"Successfully loaded new shipment to DWH for AWB: {awb} (Predicted ETA: {predicted_hours} hours)")
        else:
            conn.execute(text("""
                UPDATE fact_shipment SET
                    date_key = :date_key,
                    origin_location_key = :origin_location_key,
                    destination_location_key = :destination_location_key,
                    warehouse_key = :warehouse_key,
                    service_key = :service_key,
                    order_status = :order_status,
                    tarif_total = :tarif_total,
                    distance_km = :distance_km,
                    package_weight = :package_weight,
                    volumetric_weight = :volumetric_weight,
                    predicted_duration_hours = :predicted_duration_hours
                WHERE awb = :awb
            """), params)
            print(f"Updated shipment details in DWH for AWB: {awb} (Predicted ETA: {predicted_hours} hours)")

def handle_dispatch_assigned(awb, courier_id, vehicle_type=None):
    """Proses dispatch.assigned menggunakan data dari Kafka payload.
    ETL tidak lagi query shipping-db — vehicle_type sudah dikirim oleh shipping-service."""
    print(f"Processing dispatch.assigned event for AWB: {awb}, Courier: {courier_id}, Vehicle: {vehicle_type}")

    # Jika vehicle_type tidak ada di event (backward-compat), fallback ke shipping-db
    if not vehicle_type:
        print(f"[ETL Warning] vehicle_type kosong untuk courier {courier_id}, fallback ke query shipping-db...")
        with SHIPPING_ENGINE.connect() as src_conn:
            courier = src_conn.execute(text("SELECT vehicle_type FROM couriers WHERE id = :id"), {"id": courier_id}).mappings().first()
        vehicle_type = courier["vehicle_type"] if courier else ""

    with DWH_ENGINE.begin() as conn:
        # Ambil tarif_total dari DWH (sudah diisi saat order.created)
        tarif_total = conn.execute(text("SELECT tarif_total FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar() or 0.0

        driver_earnings = float(tarif_total) * 0.70

        # Rating ditentukan berdasarkan jenis kendaraan
        v_type = str(vehicle_type).upper()
        if "MOTOR" in v_type or "MOTORCYCLE" in v_type:
            driver_rating = 4.8
        elif "TRUCK" in v_type:
            driver_rating = 4.6
        else:
            driver_rating = 4.7

        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()
        if exists:
            conn.execute(text("""
                UPDATE fact_shipment
                SET courier_id = :courier_id,
                    driver_earnings = :driver_earnings,
                    driver_rating = :driver_rating,
                    order_status = 'PICKED_UP'
                WHERE awb = :awb
            """), {
                "courier_id": courier_id,
                "driver_earnings": driver_earnings,
                "driver_rating": driver_rating,
                "awb": awb
            })
            print(f"Updated shipment driver details (Courier: {courier_id}, Vehicle: {vehicle_type}, Earnings: {driver_earnings}) for AWB: {awb}")
        else:
            print(f"WARNING: Order AWB {awb} tidak ada di fact_shipment. Dispatch update dilewati.")

def handle_tracking_event(awb, status, location_code, event_time):
    print(f"Processing tracking event: AWB: {awb}, Status: {status}, Location: {location_code}")
    
    with DWH_ENGINE.begin() as conn:
        # Check if warehouse exists in dim_warehouse, if not fetch from warehouse DB or create stub
        wh_key = conn.execute(text("SELECT warehouse_key FROM dim_warehouse WHERE warehouse_id = :wh_id"), {"wh_id": location_code}).scalar()
        
        if not wh_key:
            # Look up from warehouse db
            with WAREHOUSE_ENGINE.connect() as wh_conn:
                wh_profile = wh_conn.execute(text("SELECT * FROM warehouses WHERE warehouse_id = :id"), {"id": location_code}).mappings().first()
            
            if wh_profile:
                conn.execute(text("""
                    INSERT INTO dim_warehouse (warehouse_id, warehouse_name, city, region, warehouse_type)
                    VALUES (:warehouse_id, :warehouse_name, :city, :region, :warehouse_type)
                """), {
                    "warehouse_id": wh_profile["warehouse_id"],
                    "warehouse_name": wh_profile["name"],
                    "city": wh_profile["city"],
                    "region": wh_profile["region"],
                    "warehouse_type": wh_profile["warehouse_type"]
                })
            else:
                # Insert a placeholder stub
                conn.execute(text("""
                    INSERT INTO dim_warehouse (warehouse_id, warehouse_name, city, region, warehouse_type)
                    VALUES (:id, :name, :city, :region, :type)
                """), {
                    "id": location_code,
                    "name": f"Transit Hub {location_code}",
                    "city": "Unknown",
                    "region": "Unknown",
                    "type": "TRANSIT"
                })
            wh_key = conn.execute(text("SELECT warehouse_key FROM dim_warehouse WHERE warehouse_id = :wh_id"), {"wh_id": location_code}).scalar()

        # Update order status and warehouse key in fact_shipment directly
        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()
        if exists:
            conn.execute(text("""
                UPDATE fact_shipment 
                SET order_status = :status,
                    warehouse_key = :wh_key 
                WHERE awb = :awb
            """), {"status": status, "wh_key": wh_key, "awb": awb})
            print(f"Updated tracking location to {location_code} and status to {status} for AWB: {awb}")
            
            if status.upper() == 'DELIVERED':
                try:
                    # Query order created_at from ORDER_ENGINE
                    with ORDER_ENGINE.connect() as src_conn:
                        created_at_val = src_conn.execute(text("SELECT created_at FROM orders WHERE awb = :awb"), {"awb": awb}).scalar()
                    
                    if created_at_val:
                        created_at = pd.to_datetime(created_at_val)
                        delivered_time = pd.to_datetime(event_time)
                        if created_at.tzinfo is not None:
                            created_at = created_at.tz_localize(None)
                        if delivered_time.tzinfo is not None:
                            delivered_time = delivered_time.tz_localize(None)
                        
                        actual_hours = (delivered_time - created_at).total_seconds() / 3600.0
                        
                        # Get predicted duration from DWH
                        predicted_hours = conn.execute(text("SELECT predicted_duration_hours FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar() or 0.0
                        error = actual_hours - float(predicted_hours)
                        
                        conn.execute(text("""
                            UPDATE fact_shipment 
                            SET actual_duration_hours = :actual_hours,
                                eta_prediction_error = :error
                            WHERE awb = :awb
                        """), {"actual_hours": actual_hours, "error": error, "awb": awb})
                        print(f"DELIVERED AWB: {awb}. Actual Duration: {actual_hours:.2f} hours. Prediction Error: {error:.2f} hours.")
                        
                        # Trigger model retraining in background thread
                        threading.Thread(target=train_model).start()
                except Exception as ex:
                    print(f"Error updating actual ML metrics on DELIVERED event: {ex}")
        else:
            print(f"WARNING: Order with AWB {awb} not found in fact_shipment. Skipping tracking update.")

def handle_notification_sent(awb, channel, event_type, success, notif_at_str):
    print(f"Processing notification event: AWB: {awb}, Channel: {channel}, Event: {event_type}, Success: {success}")
    
    notif_at = pd.to_datetime(notif_at_str)
    
    with DWH_ENGINE.begin() as conn:
        # Get or create notification type key
        notif_type_key = conn.execute(text("""
            SELECT notif_type_key FROM dim_notification_type 
            WHERE channel = :channel AND event_type = :event_type
        """), {"channel": channel, "event_type": event_type}).scalar()

        if not notif_type_key:
            conn.execute(text("""
                INSERT INTO dim_notification_type (channel, event_type)
                VALUES (:channel, :event_type)
            """), {"channel": channel, "event_type": event_type})
            notif_type_key = conn.execute(text("""
                SELECT notif_type_key FROM dim_notification_type 
                WHERE channel = :channel AND event_type = :event_type
            """), {"channel": channel, "event_type": event_type}).scalar()

        date_key = get_or_create_date_key(conn, notif_at)

        # Insert into fact_notification
        conn.execute(text("""
            INSERT INTO fact_notification (awb, date_key, notif_type_key, success, notif_at)
            VALUES (:awb, :date_key, :notif_type_key, :success, :notif_at)
        """), {
            "awb": awb,
            "date_key": date_key,
            "notif_type_key": notif_type_key,
            "success": success,
            "notif_at": notif_at
        })

        # Update notification count in fact_shipment if order exists
        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()
        if exists and success:
            conn.execute(text("""
                UPDATE fact_shipment 
                SET notification_count = notification_count + 1 
                WHERE awb = :awb
            """), {"awb": awb})
            print(f"Incremented notification_count in fact_shipment for AWB: {awb}")

# --- 3. DASHBOARD API SERVER ---
class DashboardAPIHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        # Suppress standard HTTP logs in etl-service console
        return

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()

    def handle_proxy(self, target_url, method='POST', body=None):
        import urllib.request
        import urllib.error
        
        # Ambil atau generate Correlation ID
        correlation_id = self.headers.get('X-Correlation-ID')
        if not correlation_id:
            correlation_id = f"proxy-{int(time.time() * 1000)}"
            
        headers = {
            'X-API-Key': os.getenv("API_KEY", ""),
            'X-Correlation-ID': correlation_id
        }
        if body is not None:
            headers['Content-Type'] = 'application/json'
            
        try:
            req = urllib.request.Request(
                target_url,
                data=json.dumps(body).encode('utf-8') if body else None,
                headers=headers,
                method=method
            )
            with urllib.request.urlopen(req, timeout=5) as response:
                resp_data = response.read().decode('utf-8')
                self.send_response(response.status)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                self.wfile.write(resp_data.encode('utf-8'))
        except urllib.error.HTTPError as e:
            err_data = e.read().decode('utf-8')
            self.send_response(e.code)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(err_data.encode('utf-8'))
        except Exception as e:
            self.send_response(500)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode('utf-8'))

    def do_GET(self):
        if self.path == '/api/metrics':
            try:
                metrics = self.get_dwh_metrics()
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                self.wfile.write(json.dumps(metrics).encode('utf-8'))
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                self.wfile.write(json.dumps({"error": str(e)}).encode('utf-8'))
        elif self.path.startswith('/api/proxy/tracking'):
            query = self.path.split('?')[-1] if '?' in self.path else ''
            target = f"{TRACKING_SERVICE_URL}/api/v1/tracking?{query}"
            self.handle_proxy(target, method='GET')
        elif self.path.startswith('/api/proxy/couriers/location'):
            query = self.path.split('?')[-1] if '?' in self.path else ''
            target = f"{SHIPPING_SERVICE_URL}/api/v1/couriers/location?{query}"
            self.handle_proxy(target, method='GET')
        elif self.path.startswith('/api/proxy/couriers'):
            query = self.path.split('?')[-1] if '?' in self.path else ''
            target = f"{SHIPPING_SERVICE_URL}/api/v1/couriers?{query}"
            self.handle_proxy(target, method='GET')
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        content_length = int(self.headers.get('Content-Length', 0))
        post_data = self.rfile.read(content_length) if content_length > 0 else b''
        body = json.loads(post_data.decode('utf-8')) if post_data else None

        if self.path == '/api/proxy/orders':
            self.handle_proxy(f"{ORDER_SERVICE_URL}/api/v1/orders", method='POST', body=body)
        elif self.path == '/api/proxy/tariff/calculate':
            self.handle_proxy(f"{ORDER_SERVICE_URL}/api/v1/tariff/calculate", method='POST', body=body)
        elif self.path == '/api/proxy/couriers/register':
            self.handle_proxy(f"{SHIPPING_SERVICE_URL}/api/v1/couriers/register", method='POST', body=body)
        elif self.path == '/api/proxy/dispatch':
            self.handle_proxy(f"{SHIPPING_SERVICE_URL}/api/v1/dispatch", method='POST', body=body)
        elif self.path == '/api/proxy/dispatches/confirm':
            self.handle_proxy(f"{SHIPPING_SERVICE_URL}/api/v1/dispatches/confirm", method='POST', body=body)
        elif self.path == '/api/proxy/inbound':
            self.handle_proxy(f"{WAREHOUSE_SERVICE_URL}/api/v1/inbound", method='POST', body=body)
        elif self.path == '/api/proxy/tracking/scan':
            self.handle_proxy(f"{TRACKING_SERVICE_URL}/api/v1/tracking/scan", method='POST', body=body)
        elif self.path == '/api/proxy/manifest/create':
            self.handle_proxy(f"{WAREHOUSE_SERVICE_URL}/api/v1/manifest/create", method='POST', body=body)
        elif self.path == '/api/proxy/manifest/add':
            self.handle_proxy(f"{WAREHOUSE_SERVICE_URL}/api/v1/manifest/add", method='POST', body=body)
        elif self.path == '/api/proxy/manifest/update':
            self.handle_proxy(f"{WAREHOUSE_SERVICE_URL}/api/v1/manifest/update", method='POST', body=body)
        else:
            self.send_response(404)
            self.end_headers()

    def get_dwh_metrics(self):
        data = {}
        with DWH_ENGINE.connect() as conn:
            # 1. Total Delivery & Total Revenue & Average Delivery Time
            totals = conn.execute(text("""
                SELECT 
                    COUNT(*) as total_delivery,
                    COALESCE(SUM(tarif_total), 0) as total_revenue,
                    COALESCE(AVG(actual_duration_hours), 0) as avg_duration
                FROM fact_shipment;
            """)).mappings().first()
            data['total_delivery'] = totals['total_delivery']
            data['total_revenue'] = float(totals['total_revenue'])
            data['avg_duration_hours'] = float(totals['avg_duration'])

            # 2. Notification Success Rate
            notif = conn.execute(text("""
                SELECT 
                    COUNT(*) as total_notif,
                    COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) * 100, 0) as success_rate
                FROM fact_notification;
            """)).mappings().first()
            data['notification_count'] = notif['total_notif']
            data['notification_success_rate'] = float(notif['success_rate'])

            # 3. Driver Earnings and Rating stats
            driver = conn.execute(text("""
                SELECT 
                    COALESCE(AVG(driver_earnings), 0) as avg_earnings,
                    COALESCE(AVG(driver_rating), 0) as avg_rating
                FROM fact_shipment 
                WHERE courier_id != 'N/A';
            """)).mappings().first()
            data['driver_avg_earnings'] = float(driver['avg_earnings'])
            data['driver_avg_rating'] = float(driver['avg_rating'])
            
            # Filter count to exclude N/A
            active_count = conn.execute(text("""
                SELECT COUNT(DISTINCT courier_id) FROM fact_shipment WHERE courier_id != 'N/A';
            """)).scalar() or 0
            data['driver_active_count'] = active_count

            # 4. Monthly volume trend
            monthly = conn.execute(text("""
                SELECT d.month_name, COUNT(f.shipment_key) as volume
                FROM fact_shipment f
                JOIN dim_date d ON f.date_key = d.date_key
                GROUP BY d.month_name, d.month_num
                ORDER BY d.month_num;
            """)).mappings().all()
            data['monthly_volumes'] = [int(m['volume']) for m in monthly]
            data['monthly_labels'] = [m['month_name'].strip() for m in monthly]

            if not data['monthly_volumes']:
                data['monthly_volumes'] = [0]
                data['monthly_labels'] = ['No Data']

            # 5. Service Type Distribution
            service = conn.execute(text("""
                SELECT s.service_type, COUNT(f.shipment_key) as count
                FROM fact_shipment f
                JOIN dim_service s ON f.service_key = s.service_key
                GROUP BY s.service_type;
            """)).mappings().all()
            data['service_types'] = [s['service_type'] for s in service]
            data['service_counts'] = [int(s['count']) for s in service]

            # 6. Warehouse Inbound count
            warehouse = conn.execute(text("""
                SELECT w.warehouse_id, COUNT(f.shipment_key) as count
                FROM fact_shipment f
                JOIN dim_warehouse w ON f.warehouse_key = w.warehouse_key
                GROUP BY w.warehouse_id;
            """)).mappings().all()
            data['warehouse_ids'] = [w['warehouse_id'] for w in warehouse]
            data['warehouse_counts'] = [int(w['count']) for w in warehouse]

            # 7. Notification Channel Stats
            channel_stats = conn.execute(text("""
                SELECT 
                    t.channel,
                    COALESCE(AVG(CASE WHEN f.success THEN 1.0 ELSE 0.0 END) * 100, 0) as success_rate,
                    COALESCE(AVG(CASE WHEN NOT f.success THEN 1.0 ELSE 0.0 END) * 100, 0) as failure_rate
                FROM fact_notification f
                JOIN dim_notification_type t ON f.notif_type_key = t.notif_type_key
                GROUP BY t.channel;
            """)).mappings().all()
            
            data['notif_channels'] = [c['channel'] for c in channel_stats]
            data['notif_success_rates'] = [float(c['success_rate']) for c in channel_stats]
            data['notif_failure_rates'] = [float(c['failure_rate']) for c in channel_stats]

        return data

# --- 4. MAIN KAFKA CONSUMER LOOP ---
def main():
    # Start Dashboard API server in background thread
    def run_api_server():
        try:
            server = HTTPServer(('0.0.0.0', 8085), DashboardAPIHandler)
            print("Dashboard API Server running on port 8085...")
            server.serve_forever()
        except Exception as e:
            print(f"Error running Dashboard API Server: {e}")

    threading.Thread(target=run_api_server, daemon=True).start()
    print("Starting Data Warehouse ETL Real-time Consumer...")
    
    # Run migrations/table initializations
    try:
        init_dwh_tables()
    except Exception as e:
        print(f"Error initializing DWH tables: {e}")
        time.sleep(5)
        # Retry once
        init_dwh_tables()

    # Create Kafka Consumer
    print(f"Connecting to Kafka at {KAFKA_BROKER}...")
    topics = ["papiton.events.order", "papiton.events.shipping", "papiton.events.tracking"]
    
    # Retry loop to wait for Kafka to be fully ready
    consumer = None
    for retry in range(12):
        try:
            # Mendukung comma-separated list untuk Kafka cluster multi-broker
            bootstrap_servers = [b.strip() for b in KAFKA_BROKER.split(",")]
            consumer = KafkaConsumer(
                *topics,
                bootstrap_servers=bootstrap_servers,
                auto_offset_reset='earliest',
                enable_auto_commit=True,
                group_id='dwh-etl-consumer-group',
                value_deserializer=lambda m: json.loads(m.decode('utf-8'))
            )
            print("Connected to Kafka successfully!")
            break
        except Exception as e:
            print(f"Waiting for Kafka... Attempt {retry+1}/12. Error: {e}")
            time.sleep(10)

    if not consumer:
        print("CRITICAL: Failed to connect to Kafka. Exiting.")
        return

    # Worker pool — SQLAlchemy engine sudah thread-safe dengan connection pooling
    etl_workers = int(os.getenv("ETL_WORKERS", "4"))
    executor = ThreadPoolExecutor(max_workers=etl_workers)
    print(f"Listening for events with {etl_workers} worker threads...")

    def process_message(payload, topic):
        try:
            print(f"[Worker] Received event from topic {topic}: {payload.get('awb', payload.get('resi_id', '?'))}")

            if topic == "papiton.events.order":
                awb = payload.get("awb")
                event_metadata = payload.get("metadata", {})
                if awb:
                    handle_order_created(awb, event_metadata=event_metadata)

            elif topic == "papiton.events.shipping":
                awb = payload.get("awb")
                metadata = payload.get("metadata", {})
                courier_id = metadata.get("courier_id")
                vehicle_type = metadata.get("vehicle_type", "")
                event_type = payload.get("event_type", "package.picked_up")

                if awb and courier_id:
                    handle_dispatch_assigned(awb, courier_id, vehicle_type=vehicle_type)

                if awb:
                    handle_notification_sent(
                        awb=awb,
                        channel="push",
                        event_type=event_type,
                        success=True,
                        notif_at_str=payload.get("occurred_at", datetime.datetime.now().isoformat())
                    )

            elif topic == "papiton.events.tracking":
                awb = payload.get("resi_id") or payload.get("awb")
                status = payload.get("activity_code") or payload.get("event_type") or "IN_TRANSIT"
                if isinstance(status, str) and status.startswith("package."):
                    status = status.replace("package.", "").upper()

                metadata = payload.get("metadata", {})
                location_code = payload.get("location_code") or metadata.get("location") or "WH-BDG"
                if isinstance(location_code, str) and location_code.startswith("Warehouse "):
                    location_code = location_code.replace("Warehouse ", "")

                event_time = payload.get("timestamp") or payload.get("occurred_at") or datetime.datetime.now().isoformat()

                if awb:
                    handle_tracking_event(awb, status, location_code, event_time)
                    handle_notification_sent(
                        awb=awb,
                        channel="email",
                        event_type=f"package.{status.lower()}",
                        success=True,
                        notif_at_str=event_time
                    )

        except Exception as e:
            print(f"[Worker] Error handling message from {topic}: {e}")

    for message in consumer:
        executor.submit(process_message, message.value, message.topic)

if __name__ == "__main__":
    main()
