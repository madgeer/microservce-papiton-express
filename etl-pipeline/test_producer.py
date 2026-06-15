import os
import json
import time
import datetime
from sqlalchemy import create_engine, text
from kafka import KafkaProducer

# --- CONFIGURATIONS ---
KAFKA_BROKER = os.getenv("KAFKA_BROKER", "localhost:9092")
ORDER_DB_URL = os.getenv("ORDER_DB_URL", "postgresql://postgres:admin123@localhost:5434/papiton_order_tariff_service_db")
SHIPPING_DB_URL = os.getenv("SHIPPING_DB_URL", "postgresql://user:password@localhost:5433/shipping_test_db")
WAREHOUSE_DB_URL = os.getenv("WAREHOUSE_DB_URL", "postgresql://postgres:postgres@localhost:5432/papiton_warehouse")

print(f"Connecting to Kafka at {KAFKA_BROKER} and DBs...")
try:
    producer = KafkaProducer(
        bootstrap_servers=[KAFKA_BROKER],
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )
    
    order_engine = create_engine(ORDER_DB_URL)
    shipping_engine = create_engine(SHIPPING_DB_URL)
    warehouse_engine = create_engine(WAREHOUSE_DB_URL)
    print("Connections established!")
except Exception as e:
    print(f"Error establishing connections: {e}")
    print("NOTE: Make sure docker-compose is running and ports 9092, 5434, 5433, 5432 are exposed to localhost.")
    exit(1)

def test_pipeline():
    test_awb = f"RESI-TEST-{int(time.time())}"
    print(f"\n=== Running Test for AWB: {test_awb} ===")

    # 1. POPULATE SOURCE DATABASES
    print("Step 1: Populating mock data in operational databases...")
    
    # 1.1 Warehouse DB DDL and Mock Data
    with warehouse_engine.begin() as conn:
        # Create tables if they do not exist
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS warehouses (
                warehouse_id VARCHAR(50) PRIMARY KEY,
                name VARCHAR(100) NOT NULL,
                city VARCHAR(50) NOT NULL,
                region VARCHAR(50) NOT NULL,
                warehouse_type VARCHAR(20) NOT NULL
            );
        """))
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS inbound_packages (
                resi VARCHAR(50) PRIMARY KEY,
                warehouse_id VARCHAR(50) NOT NULL REFERENCES warehouses(warehouse_id),
                status VARCHAR(50) NOT NULL,
                is_express BOOLEAN DEFAULT FALSE,
                special_handling TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        """))
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS manifests (
                manifest_id VARCHAR(50) PRIMARY KEY,
                truck_id VARCHAR(50) NOT NULL,
                driver_name VARCHAR(100) NOT NULL,
                status VARCHAR(50) NOT NULL DEFAULT 'CREATED',
                origin_warehouse VARCHAR(50) REFERENCES warehouses(warehouse_id),
                destination_warehouse VARCHAR(50) REFERENCES warehouses(warehouse_id),
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        """))
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS manifest_packages (
                manifest_id VARCHAR(50) REFERENCES manifests(manifest_id) ON DELETE CASCADE,
                resi VARCHAR(50) REFERENCES inbound_packages(resi) ON DELETE CASCADE,
                PRIMARY KEY (manifest_id, resi)
            );
        """))
        # Create warehouse if not exists
        conn.execute(text("""
            INSERT INTO warehouses (warehouse_id, name, city, region, warehouse_type)
            VALUES ('WH-BDG', 'Hub Utama Bandung', 'Bandung', 'Jawa Barat', 'HUB')
            ON CONFLICT (warehouse_id) DO NOTHING;
        """))
        # Insert inbound package
        conn.execute(text("""
            INSERT INTO inbound_packages (resi, warehouse_id, status)
            VALUES (:resi, 'WH-BDG', 'INBOUND_COMPLETED')
            ON CONFLICT (resi) DO NOTHING;
        """), {"resi": test_awb})

    # 1.2 Order DB
    with order_engine.begin() as conn:
        # Patch the table if it already exists but lacks the new column
        conn.execute(text("""
            ALTER TABLE orders ADD COLUMN IF NOT EXISTS volumetric_weight DOUBLE PRECISION DEFAULT 0.0;
        """))
        
        conn.execute(text("""
            INSERT INTO orders (
                awb, sender_name, sender_phone, sender_email, sender_address, sender_city, sender_lat, sender_lng,
                recipient_name, recipient_phone, recipient_email, recipient_address, recipient_city, recipient_lat, recipient_lng,
                package_length, package_width, package_height, package_weight, volumetric_weight,
                service_type, has_insurance, has_packing,
                tarif_total, distance, eta, status, created_at
            ) VALUES (
                :awb, 'Budi', '0812', 'budi@test.com', 'Sarijadi', 'Bandung', -6.1, 107.2,
                'Andi', '0813', 'andi@test.com', 'Sudirman', 'Jakarta', -6.2, 106.8,
                30, 20, 10, 2.5, 1.0,
                'EXPRESS', TRUE, FALSE,
                25000, 150.5, '2 Hours', 'CREATED', NOW()
            ) ON CONFLICT (awb) DO NOTHING;
        """), {"awb": test_awb})

    # 1.3 Shipping DB
    with shipping_engine.begin() as conn:
        # Insert Courier
        conn.execute(text("""
            INSERT INTO couriers (id, name, phone_number, zone, status, vehicle_type)
            VALUES ('C-TEST-01', 'Asep Sukandar', '081234', 'Bandung', 'AVAILABLE', 'MOTORCYCLE')
            ON CONFLICT (id) DO NOTHING;
        """))
        # Insert Dispatch
        conn.execute(text("""
            INSERT INTO dispatches (id, order_id, courier_id, status, route_instruction)
            VALUES (:id, :awb, 'C-TEST-01', 'ASSIGNED', 'Go to warehouse BDG')
            ON CONFLICT (id) DO NOTHING;
        """), {"id": f"DSP-{test_awb}", "awb": test_awb})

    print("Operational DBs populated successfully!")

    # 2. SEND KAFKA EVENTS
    # 2.1 Order Created Event
    print("\nStep 2: Publishing papiton.events.order (order.created)...")
    order_event = {
        "event_id": f"EVT-ORD-{test_awb}",
        "event_type": "order.created",
        "user_id": "customer@test.com",
        "awb": test_awb,
        "occurred_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "metadata": {
            "status": "CREATED"
        },
        # Backward compatibility
        "email": "customer@test.com",
        "status": "CREATED",
        "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    }
    producer.send("papiton.events.order", order_event)
    producer.flush()
    time.sleep(2) # Give consumer time to process

    # 2.2 Shipping Dispatch Assigned Event
    print("Step 3: Publishing papiton.events.shipping (package.picked_up / dispatch.assigned)...")
    shipping_event = {
        "event_id": f"EVT-DSP-{test_awb}",
        "event_type": "package.picked_up",
        "user_id": "customer@test.com",
        "awb": test_awb,
        "occurred_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "metadata": {
            "courier_id": "C-TEST-01",
            "status": "dispatch.assigned",
            "route_instruction": "Go to warehouse BDG"
        }
    }
    producer.send("papiton.events.shipping", shipping_event)
    producer.flush()
    time.sleep(2)

    # 2.3 Tracking Inbound/Transit Event
    print("Step 4: Publishing papiton.events.tracking (package.in_transit)...")
    tracking_event = {
        "event_id": f"EVT-INB-{test_awb}",
        "event_type": "package.in_transit",
        "user_id": "customer@test.com",
        "awb": test_awb,
        "occurred_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "metadata": {
            "location": "Warehouse WH-UPI",
            "status": "inventory.inbound"
        },
        # Backward compatibility
        "resi_id": test_awb,
        "location_code": "WH-UPI",
        "activity_code": "IN_TRANSIT",
        "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    }
    producer.send("papiton.events.tracking", tracking_event)
    producer.flush()
    print("All events published successfully!")
    print("\nCheck the etl-service log to see the real-time consumption output!")

if __name__ == "__main__":
    test_pipeline()
