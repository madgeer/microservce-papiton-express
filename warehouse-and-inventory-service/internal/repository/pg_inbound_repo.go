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
	_, err := r.db.Exec(`
		INSERT INTO inbound_packages (resi, warehouse_id, status, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (resi)
		DO UPDATE SET status = EXCLUDED.status, warehouse_id = EXCLUDED.warehouse_id, updated_at = EXCLUDED.updated_at
	`, resi, warehouseID, status)
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
