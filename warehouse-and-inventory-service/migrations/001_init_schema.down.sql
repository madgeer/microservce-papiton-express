-- File: migrations/001_init_schema.down.sql
-- Digunakan untuk me-rollback (menghapus) tabel jika ada kesalahan

DROP TABLE IF EXISTS sorting_lanes CASCADE;
DROP TABLE IF EXISTS manifest_packages CASCADE;
DROP TABLE IF EXISTS manifests CASCADE;
DROP TABLE IF EXISTS inbound_packages CASCADE;
