import os
import json
import datetime
import time
import pandas as pd
from sqlalchemy import create_engine, text
from pymongo import MongoClient
from kafka import KafkaConsumer

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
            CREATE TABLE IF NOT EXISTS dim_order (
                order_key SERIAL PRIMARY KEY,
                awb VARCHAR(50) UNIQUE NOT NULL,
                sender_name VARCHAR(100),
                sender_city VARCHAR(50),
                recipient_city VARCHAR(50),
                eta TIMESTAMP,
                order_status VARCHAR(50),
                created_at TIMESTAMP
            );
        """))
        
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS dim_courier (
                courier_key SERIAL PRIMARY KEY,
                courier_id VARCHAR(50) UNIQUE NOT NULL,
                courier_name VARCHAR(100) NOT NULL,
                zone VARCHAR(50),
                vehicle_type VARCHAR(50),
                dispatch_status VARCHAR(50)
            );
        """))
        
        # Sentinel Row for dim_courier
        conn.execute(text("""
            INSERT INTO dim_courier (courier_key, courier_id, courier_name, zone, vehicle_type, dispatch_status)
            VALUES (-1, 'N/A', 'Belum Ditugaskan', 'N/A', 'N/A', 'N/A')
            ON CONFLICT (courier_id) DO NOTHING;
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
                date_key INT NOT NULL REFERENCES dim_date(date_key),
                order_key INT NOT NULL REFERENCES dim_order(order_key),
                courier_key INT NOT NULL REFERENCES dim_courier(courier_key),
                warehouse_key INT NOT NULL REFERENCES dim_warehouse(warehouse_key),
                service_key INT NOT NULL REFERENCES dim_service(service_key),
                tarif_total NUMERIC NOT NULL,
                distance_km NUMERIC NOT NULL,
                package_weight NUMERIC NOT NULL,
                volumetric_weight NUMERIC NOT NULL,
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

        # Load dim_order
        conn.execute(text("""
            INSERT INTO dim_order (awb, sender_name, sender_city, recipient_city, eta, order_status, created_at)
            VALUES (:awb, :sender_name, :sender_city, :recipient_city, :eta, :order_status, :created_at)
            ON CONFLICT (awb) DO UPDATE SET
                order_status = EXCLUDED.order_status,
                eta = EXCLUDED.eta;
        """), {
            "awb": order["awb"],
            "sender_name": order["sender_name"],
            "sender_city": order["sender_city"],
            "recipient_city": order["recipient_city"],
            "eta": pd.to_datetime(order["eta"]) if pd.notnull(order["eta"]) else None,
            "order_status": order["status"],
            "created_at": pd.to_datetime(order["created_at"])
        })

        # Get Keys
        created_at = pd.to_datetime(order["created_at"])
        date_key = get_or_create_date_key(conn, created_at)
        
        order_key = conn.execute(text("SELECT order_key FROM dim_order WHERE awb = :awb"), {"awb": awb}).scalar()
        warehouse_key = conn.execute(text("SELECT warehouse_key FROM dim_warehouse WHERE warehouse_id = :warehouse_id"), {"warehouse_id": warehouse_id}).scalar()
        service_key = conn.execute(text("SELECT service_key FROM dim_service WHERE service_type = :service_type"), {"service_type": order["service_type"]}).scalar()
        
        # Calculate Volumetric Weight
        volumetric_weight = (order["package_length"] * order["package_width"] * order["package_height"]) / 6000.0

        # Idempotent load to fact_shipment
        exists = conn.execute(text("SELECT shipment_key FROM fact_shipment WHERE order_key = :order_key"), {"order_key": order_key}).scalar()
        
        params = {
            "date_key": date_key,
            "order_key": order_key,
            "courier_key": -1, # Default sentinel
            "warehouse_key": warehouse_key,
            "service_key": service_key,
            "tarif_total": order["tarif_total"],
            "distance_km": order["distance"],
            "package_weight": order["package_weight"],
            "volumetric_weight": volumetric_weight
        }

        if not exists:
            conn.execute(text("""
                INSERT INTO fact_shipment (date_key, order_key, courier_key, warehouse_key, service_key, tarif_total, distance_km, package_weight, volumetric_weight)
                VALUES (:date_key, :order_key, :courier_key, :warehouse_key, :service_key, :tarif_total, :distance_km, :package_weight, :volumetric_weight)
            """), params)
            print(f"Successfully loaded new shipment to DWH for AWB: {awb}")
        else:
            conn.execute(text("""
                UPDATE fact_shipment SET
                    date_key = :date_key,
                    warehouse_key = :warehouse_key,
                    service_key = :service_key,
                    tarif_total = :tarif_total,
                    distance_km = :distance_km,
                    package_weight = :package_weight,
                    volumetric_weight = :volumetric_weight
                WHERE order_key = :order_key
            """), params)
            print(f"Updated shipment details in DWH for AWB: {awb}")

def handle_dispatch_assigned(awb, courier_id):
    print(f"Processing dispatch.assigned event for AWB: {awb}, Courier: {courier_id}")
    # Extract courier details from shipping-db
    with SHIPPING_ENGINE.connect() as src_conn:
        courier = src_conn.execute(text("SELECT * FROM couriers WHERE id = :id"), {"id": courier_id}).mappings().first()

    if not courier:
        print(f"WARNING: Courier with ID {courier_id} not found in Shipping DB.")
        return

    with DWH_ENGINE.begin() as conn:
        # Load dim_courier
        conn.execute(text("""
            INSERT INTO dim_courier (courier_id, courier_name, zone, vehicle_type, dispatch_status)
            VALUES (:courier_id, :courier_name, :zone, :vehicle_type, :dispatch_status)
            ON CONFLICT (courier_id) DO UPDATE SET
                courier_name = EXCLUDED.courier_name,
                zone = EXCLUDED.zone,
                vehicle_type = EXCLUDED.vehicle_type,
                dispatch_status = EXCLUDED.dispatch_status;
        """), {
            "courier_id": courier["id"],
            "courier_name": courier["name"],
            "zone": courier["zone"],
            "vehicle_type": courier["vehicle_type"],
            "dispatch_status": courier["status"]
        })

        # Fetch keys
        order_key = conn.execute(text("SELECT order_key FROM dim_order WHERE awb = :awb"), {"awb": awb}).scalar()
        courier_key = conn.execute(text("SELECT courier_key FROM dim_courier WHERE courier_id = :courier_id"), {"courier_id": courier_id}).scalar()

        if order_key:
            # Update fact_shipment
            conn.execute(text("""
                UPDATE fact_shipment 
                SET courier_key = :courier_key 
                WHERE order_key = :order_key
            """), {"courier_key": courier_key, "order_key": order_key})
            
            # Update dim_order status
            conn.execute(text("""
                UPDATE dim_order 
                SET order_status = 'PICKED_UP' 
                WHERE order_key = :order_key
            """), {"order_key": order_key})
            print(f"Updated shipment courier to {courier['name']} for AWB: {awb}")
        else:
            print(f"WARNING: Order with AWB {awb} not found in dim_order. Cannot assign courier yet.")

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

        # Update order status and warehouse in DWH
        order_key = conn.execute(text("SELECT order_key FROM dim_order WHERE awb = :awb"), {"awb": awb}).scalar()
        if order_key:
            conn.execute(text("""
                UPDATE dim_order 
                SET order_status = :status 
                WHERE order_key = :order_key
            """), {"status": status, "order_key": order_key})

            conn.execute(text("""
                UPDATE fact_shipment 
                SET warehouse_key = :wh_key 
                WHERE order_key = :order_key
            """), {"wh_key": wh_key, "order_key": order_key})
            print(f"Updated tracking location to {location_code} for AWB: {awb}")
        else:
            print(f"WARNING: Order with AWB {awb} not found in dim_order. Skipping tracking update.")

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
        order_key = conn.execute(text("SELECT order_key FROM dim_order WHERE awb = :awb"), {"awb": awb}).scalar()
        if order_key and success:
            conn.execute(text("""
                UPDATE fact_shipment 
                SET notification_count = notification_count + 1 
                WHERE order_key = :order_key
            """), {"order_key": order_key})
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
                # Typical payload from tracking: { "resi_id": "...", "location_code": "...", "activity_code": "...", "timestamp": "..." }
                awb = payload.get("resi_id")
                status = payload.get("activity_code", "IN_TRANSIT")
                location_code = payload.get("location_code", "WH-BDG")
                event_time = payload.get("timestamp", datetime.datetime.now().isoformat())
                
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
