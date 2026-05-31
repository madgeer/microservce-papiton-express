-- File: migrations/001_init_schema.up.sql
-- Digunakan untuk membuat struktur tabel di PostgreSQL

CREATE TABLE inbound_packages (
    resi VARCHAR(50) PRIMARY KEY,
    warehouse_id VARCHAR(50) NOT NULL,
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
    origin_warehouse VARCHAR(50),
    destination_warehouse VARCHAR(50),
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
    warehouse_id VARCHAR(50) NOT NULL,
    priority_type VARCHAR(20) NOT NULL, -- Contoh: 'EXPRESS' atau 'REGULAR'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
