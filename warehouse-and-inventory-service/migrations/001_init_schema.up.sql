-- File: migrations/001_init_schema.up.sql
-- Digunakan untuk membuat struktur tabel di PostgreSQL

CREATE TABLE warehouses (
    warehouse_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    city VARCHAR(50) NOT NULL,
    region VARCHAR(50) NOT NULL,
    warehouse_type VARCHAR(20) NOT NULL
);

INSERT INTO warehouses (warehouse_id, name, city, region, warehouse_type) VALUES
('WH-BDG', 'Hub Utama Bandung', 'Bandung', 'Jawa Barat', 'HUB'),
('WH-JKT', 'Hub Regional Jakarta', 'Jakarta', 'DKI Jakarta', 'HUB'),
('WH-SUB', 'Hub Timur Surabaya', 'Surabaya', 'Jawa Timur', 'HUB'),
('WH-SMG', 'Hub Tengah Semarang', 'Semarang', 'Jawa Tengah', 'HUB'),
('WH-MES', 'Hub Sumatera Medan', 'Medan', 'Sumatera Utara', 'REGIONAL'),
('WH-UPG', 'Hub Sulawesi Makassar', 'Makassar', 'Sulawesi Selatan', 'REGIONAL'),
('WH-YRK', 'Transit Yogyakarta', 'Yogyakarta', 'DI Yogyakarta', 'TRANSIT'),
('WH-DPS', 'Transit Denpasar', 'Denpasar', 'Bali', 'TRANSIT'),
('WH-UPI', 'Hub Transit UPI', 'Bandung', 'Jawa Barat', 'TRANSIT');

CREATE TABLE inbound_packages (
    resi VARCHAR(50) PRIMARY KEY,
    warehouse_id VARCHAR(50) NOT NULL REFERENCES warehouses(warehouse_id),
    status VARCHAR(50) NOT NULL,
    is_express BOOLEAN DEFAULT FALSE,
    special_handling TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE manifests (
    manifest_id VARCHAR(50) PRIMARY KEY,
    truck_id VARCHAR(50) NOT NULL,
    driver_name VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'CREATED',
    origin_warehouse VARCHAR(50) REFERENCES warehouses(warehouse_id),
    destination_warehouse VARCHAR(50) REFERENCES warehouses(warehouse_id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE manifest_packages (
    manifest_id VARCHAR(50) REFERENCES manifests(manifest_id) ON DELETE CASCADE,
    resi VARCHAR(50) REFERENCES inbound_packages(resi) ON DELETE CASCADE,
    PRIMARY KEY (manifest_id, resi)
);

CREATE TABLE sorting_lanes (
    lane_id VARCHAR(50) PRIMARY KEY,
    warehouse_id VARCHAR(50) NOT NULL REFERENCES warehouses(warehouse_id),
    priority_type VARCHAR(20) NOT NULL, -- Contoh: 'EXPRESS' atau 'REGULAR'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
