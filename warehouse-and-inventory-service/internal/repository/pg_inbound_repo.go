package repository

import (
	"database/sql"
	"strings"
)

/*
* PostgresInboundRepo adalah implementasi InboundRepository menggunakan PostgreSQL.
 */
type PostgresInboundRepo struct {
	db *sql.DB
}

/* NewPostgresInboundRepo membuat instance baru untuk PostgresInboundRepo */
func NewPostgresInboundRepo(db *sql.DB) *PostgresInboundRepo {
	return &PostgresInboundRepo{db: db}
}

func (r *PostgresInboundRepo) UpdateStockStatus(resi string, warehouseID string, status string) error {
	_, err := r.db.Exec("UPDATE inbound_packages SET status = $1 WHERE resi = $2", status, resi)
	return err
}

func (r *PostgresInboundRepo) GetItemByResi(resi string) (string, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM inbound_packages WHERE resi = $1", resi).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (r *PostgresInboundRepo) UpdatePackageMetadata(resi string, instructions []string) error {
	specialHandling := strings.Join(instructions, ",")
	_, err := r.db.Exec("UPDATE inbound_packages SET special_handling = $1 WHERE resi = $2", specialHandling, resi)
	return err
}
