package repository

import (
	"database/sql"
)

/*
* PostgresManifestRepo adalah implementasi ManifestRepository menggunakan PostgreSQL.
 */
type PostgresManifestRepo struct {
	db *sql.DB
}

/* NewPostgresManifestRepo membuat instance baru untuk PostgresManifestRepo */
func NewPostgresManifestRepo(db *sql.DB) *PostgresManifestRepo {
	return &PostgresManifestRepo{db: db}
}

func (r *PostgresManifestRepo) CreateManifest(truckID string, driverName string) (string, error) {
	var id string
	// Menggunakan random string generator SQL untuk mensimulasikan ID manifest baru
	err := r.db.QueryRow("INSERT INTO manifests (manifest_id, truck_id, driver_name, status) VALUES ('MNF-' || CAST(floor(random() * 1000000) AS VARCHAR), $1, $2, 'CREATED') RETURNING manifest_id AS id", truckID, driverName).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *PostgresManifestRepo) AddPackageToManifest(manifestID string, resi string) error {
	_, err := r.db.Exec("INSERT INTO manifest_packages (manifest_id, resi) VALUES ($1, $2)", manifestID, resi)
	return err
}

func (r *PostgresManifestRepo) UpdateManifestStatus(manifestID string, status string) error {
	_, err := r.db.Exec("UPDATE manifests SET status = $1 WHERE manifest_id = $2", status, manifestID)
	return err
}

func (r *PostgresManifestRepo) GetManifestStatus(manifestID string) (string, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM manifests WHERE manifest_id = $1", manifestID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}
