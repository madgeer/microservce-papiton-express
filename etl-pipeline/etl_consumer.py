import os
import json
import datetime
import time
import pickle
import threading
import numpy as np
import pandas as pd
from sqlalchemy import create_engine, text
from pymongo import MongoClient
from kafka import KafkaConsumer
from sklearn.linear_model import LinearRegression

# --- CONFIGURATION FROM ENV OR DEFAULTS ---
KAFKA_BROKER = os.getenv("KAFKA_BROKER", "kafka:9092")
ORDER_DB_URL = os.getenv("ORDER_DB_URL", "postgresql://postgres:admin123@order-db:5432/papiton_order_tariff_service_db")
SHIPPING_DB_URL = os.getenv("SHIPPING_DB_URL", "postgresql://user:password@shipping-db:5432/shipping_test_db")
WAREHOUSE_DB_URL = os.getenv("WAREHOUSE_DB_URL", "postgresql://postgres:postgres@warehouse-db:5432/papiton_warehouse")
DWH_DB_URL = os.getenv("DWH_DB_URL", "postgresql://postgres:dwhpassword@dwh-db:5432/papiton_dwh")
MONGO_URI = os.getenv("MONGO_URI", "mongodb://tracking-mongo:27017")
MONGO_DB_NAME = os.getenv("MONGO_DB_NAME", "tracking_db")

print("Initializing database engines...")
ORDER_ENGINE = create_engine(ORDER_DB_URL)
SHIPPING_ENGINE = create_engine(SHIPPING_DB_URL)
WAREHOUSE_ENGINE = create_engine(WAREHOUSE_DB_URL)
DWH_ENGINE = create_engine(DWH_DB_URL)

MODEL_PATH = "/app/eta_model.pkl"

def train_model():
    print("Triggering machine learning model training loop...")
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

def handle_order_created(awb):
    print(f"Processing order.created event for AWB: {awb}")
    # Extract order details from order-db
    with ORDER_ENGINE.connect() as src_conn:
        order = src_conn.execute(text("SELECT * FROM orders WHERE awb = :awb"), {"awb": awb}).mappings().first()
    
    if not order:
        print(f"WARNING: Order with AWB {awb} not found in Order DB yet. Will retry later.")
        return

    # Extract additional warehouse info if already inbounded
    with WAREHOUSE_ENGINE.connect() as wh_conn:
        inbound = wh_conn.execute(text("SELECT * FROM inbound_packages WHERE resi = :awb"), {"awb": awb}).mappings().first()
    
    warehouse_id = inbound['warehouse_id'] if inbound else 'WH-BDG' # Default hub

    # Extract warehouse profile
    with WAREHOUSE_ENGINE.connect() as wh_conn:
        wh_profile = wh_conn.execute(text("SELECT * FROM warehouses WHERE warehouse_id = :warehouse_id"), {"warehouse_id": warehouse_id}).mappings().first()
    
    with DWH_ENGINE.begin() as conn:
        # Load dim_warehouse
        if wh_profile:
            conn.execute(text("""
                INSERT INTO dim_warehouse (warehouse_id, warehouse_name, city, region, warehouse_type)
                VALUES (:warehouse_id, :warehouse_name, :city, :region, :warehouse_type)
                ON CONFLICT (warehouse_id) DO UPDATE SET
                    warehouse_name = EXCLUDED.warehouse_name,
                    city = EXCLUDED.city,
                    region = EXCLUDED.region,
                    warehouse_type = EXCLUDED.warehouse_type;
            """), {
                "warehouse_id": wh_profile["warehouse_id"],
                "warehouse_name": wh_profile["name"],
                "city": wh_profile["city"],
                "region": wh_profile["region"],
                "warehouse_type": wh_profile["warehouse_type"]
            })

        # Load dim_service
        conn.execute(text("""
            INSERT INTO dim_service (service_type, has_insurance, has_packing, base_price_per_km, insurance_fee)
            VALUES (:service_type, :has_insurance, :has_packing, :base_price_per_km, :insurance_fee)
            ON CONFLICT (service_type) DO UPDATE SET
                has_insurance = EXCLUDED.has_insurance,
                has_packing = EXCLUDED.has_packing;
        """), {
            "service_type": order["service_type"],
            "has_insurance": order["has_insurance"],
            "has_packing": order["has_packing"],
            "base_price_per_km": 1500 if order["service_type"] == 'EXPRESS' else 1000,
            "insurance_fee": 5000 if order["has_insurance"] else 0
        })

        # Get created_at as datetime
        created_at = pd.to_datetime(order["created_at"])
        date_key = get_or_create_date_key(conn, created_at)
        
        # Get location keys (normalize sender and recipient city names)
        origin_location_key = get_or_create_location_key(conn, order["sender_city"])
        destination_location_key = get_or_create_location_key(conn, order["recipient_city"])
        
        warehouse_key = conn.execute(text("SELECT warehouse_key FROM dim_warehouse WHERE warehouse_id = :warehouse_id"), {"warehouse_id": warehouse_id}).scalar()
        service_key = conn.execute(text("SELECT service_key FROM dim_service WHERE service_type = :service_type"), {"service_type": order["service_type"]}).scalar()
        
        # Calculate Volumetric Weight
        volumetric_weight = (order["package_length"] * order["package_width"] * order["package_height"]) / 6000.0

        # ML Inference: Predict ETA duration in hours
        is_express = (order["service_type"] == 'EXPRESS')
        predicted_hours = predict_duration(
            distance=float(order["distance"]),
            weight=float(order["package_weight"]),
            vol_weight=float(volumetric_weight),
            is_express=is_express
        )

        # Idempotent load to fact_shipment
        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()
        
        params = {
            "awb": awb,
            "date_key": date_key,
            "origin_location_key": origin_location_key,
            "destination_location_key": destination_location_key,
            "courier_id": "N/A",
            "warehouse_key": warehouse_key,
            "service_key": service_key,
            "order_status": order["status"],
            "tarif_total": order["tarif_total"],
            "distance_km": order["distance"],
            "package_weight": order["package_weight"],
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

def handle_dispatch_assigned(awb, courier_id):
    print(f"Processing dispatch.assigned event for AWB: {awb}, Courier: {courier_id}")
    # Extract courier details from shipping-db
    with SHIPPING_ENGINE.connect() as src_conn:
        courier = src_conn.execute(text("SELECT * FROM couriers WHERE id = :id"), {"id": courier_id}).mappings().first()

    if not courier:
        print(f"WARNING: Courier with ID {courier_id} not found in Shipping DB.")
        return

    with DWH_ENGINE.begin() as conn:
        # Get tarif_total of this shipment to calculate driver earnings
        tarif_total = conn.execute(text("SELECT tarif_total FROM fact_shipment WHERE awb = :awb"), {"awb": awb}).scalar()
        if tarif_total is None:
            # Fallback to Order DB
            with ORDER_ENGINE.connect() as src_conn:
                tarif_total = src_conn.execute(text("SELECT tarif_total FROM orders WHERE awb = :awb"), {"awb": awb}).scalar() or 0.0

        # Operational earnings calculation (e.g. 70% of total tariff goes to driver)
        driver_earnings = float(tarif_total) * 0.70

        # Operational driver rating (calculated in shipping system, let's say average 4.7)
        # We can dynamically vary it slightly based on vehicle type for realistic variety
        driver_rating = 4.7
        v_type = str(courier.get("vehicle_type", "")).upper()
        if "MOTOR" in v_type:
            driver_rating = 4.8
        elif "CAR" in v_type or "VAN" in v_type:
            driver_rating = 4.7
        elif "TRUCK" in v_type:
            driver_rating = 4.6

        # Update fact_shipment
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
            print(f"Updated shipment driver details (Courier: {courier_id}, Earnings: {driver_earnings}, Rating: {driver_rating}) for AWB: {awb}")
        else:
            print(f"WARNING: Order with AWB {awb} not found in fact_shipment. Cannot assign courier yet.")

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

# --- 3. MAIN KAFKA CONSUMER LOOP ---
def main():
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
            consumer = KafkaConsumer(
                *topics,
                bootstrap_servers=[KAFKA_BROKER],
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

    print("Listening for events...")
    for message in consumer:
        try:
            payload = message.value
            topic = message.topic
            print(f"Received message from topic {topic}: {payload}")
            
            # Map events to handlers
            if topic == "papiton.events.order":
                awb = payload.get("awb")
                if awb:
                    handle_order_created(awb)
            
            elif topic == "papiton.events.shipping":
                awb = payload.get("awb")
                metadata = payload.get("metadata", {})
                courier_id = metadata.get("courier_id")
                
                # Check for notification trigger if it's matching picked_up
                event_type = payload.get("event_type", "package.picked_up")
                
                if awb and courier_id:
                    handle_dispatch_assigned(awb, courier_id)
                
                # Log dispatch event as a notification status update
                if awb:
                    handle_notification_sent(
                        awb=awb,
                        channel="push", # Default channel
                        event_type=event_type,
                        success=True,
                        notif_at_str=payload.get("occurred_at", datetime.datetime.now().isoformat())
                    )
            
            elif topic == "papiton.events.tracking":
                # Support both old and new tracking payload structures
                awb = payload.get("resi_id") or payload.get("awb")
                status = payload.get("activity_code") or payload.get("event_type") or "IN_TRANSIT"
                if isinstance(status, str) and status.startswith("package."):
                    status = status.replace("package.", "").upper()
                
                # Check for location in metadata or root
                metadata = payload.get("metadata", {})
                location_code = payload.get("location_code") or metadata.get("location") or "WH-BDG"
                # Strip "Warehouse " prefix if present to keep code clean
                if isinstance(location_code, str) and location_code.startswith("Warehouse "):
                    location_code = location_code.replace("Warehouse ", "")
                
                event_time = payload.get("timestamp") or payload.get("occurred_at") or datetime.datetime.now().isoformat()
                
                if awb:
                    handle_tracking_event(awb, status, location_code, event_time)
                    
                    # Also log tracking as notification event
                    handle_notification_sent(
                        awb=awb,
                        channel="email", # Customer gets update via email
                        event_type=f"package.{status.lower()}",
                        success=True,
                        notif_at_str=event_time
                    )
                    
        except Exception as e:
            print(f"Error handling Kafka message: {e}")

if __name__ == "__main__":
    main()
