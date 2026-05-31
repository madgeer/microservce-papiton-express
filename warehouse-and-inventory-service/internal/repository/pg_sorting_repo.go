package repository

import (
	"database/sql"
)

/*
* PostgresSortingRepo adalah implementasi SortingRepository menggunakan PostgreSQL.
 */
type PostgresSortingRepo struct {
	db *sql.DB
}

/* NewPostgresSortingRepo membuat instance baru untuk PostgresSortingRepo */
func NewPostgresSortingRepo(db *sql.DB) *PostgresSortingRepo {
	return &PostgresSortingRepo{db: db}
}

func (r *PostgresSortingRepo) AssignToLane(resi string, laneID string) error {
	_, err := r.db.Exec("UPDATE inbound_packages SET lane_id = $1 WHERE resi = $2", laneID, resi)
	return err
}

func (r *PostgresSortingRepo) GetPackagesInLane(laneID string) ([]string, error) {
	rows, err := r.db.Query("SELECT resi FROM inbound_packages WHERE lane_id = $1", laneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resis []string
	for rows.Next() {
		var resi string
		if err := rows.Scan(&resi); err != nil {
			return nil, err
		}
		resis = append(resis, resi)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return resis, nil
}
