package postgres

import (
	"context"
	"database/sql"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
)

type dispatchRepository struct {
	db *sql.DB
}

/* NewDispatchRepository membuat instance baru dari DispatchRepository menggunakan PostgreSQL */
func NewDispatchRepository(db *sql.DB) domain.DispatchRepository {
	return &dispatchRepository{db: db}
}

func (r *dispatchRepository) Create(ctx context.Context, d *domain.Dispatch) error {
	// Pastikan tabel dispatches ada di database
	queryCreateTable := `
	CREATE TABLE IF NOT EXISTS dispatches (
		id VARCHAR(50) PRIMARY KEY,
		order_id VARCHAR(50) NOT NULL,
		courier_id VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL,
		route_instruction TEXT NOT NULL
	);`

	_, err := r.db.ExecContext(ctx, queryCreateTable)
	if err != nil {
		return err
	}

	queryInsert := `
	INSERT INTO dispatches (id, order_id, courier_id, status, route_instruction)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO UPDATE SET 
		status = EXCLUDED.status,
		route_instruction = EXCLUDED.route_instruction;`

	_, err = r.db.ExecContext(ctx, queryInsert, d.ID, d.OrderID, d.CourierID, string(d.Status), d.RouteInstruction)
	return err
}

func (r *dispatchRepository) UpdateStatus(ctx context.Context, id string, status domain.DispatchStatus) error {
	queryUpdate := `UPDATE dispatches SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, queryUpdate, string(status), id)
	return err
}

func (r *dispatchRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Dispatch, error) {
	querySelect := `SELECT id, order_id, courier_id, status, route_instruction FROM dispatches WHERE order_id = $1`

	var d domain.Dispatch
	var statusStr string
	err := r.db.QueryRowContext(ctx, querySelect, orderID).Scan(&d.ID, &d.OrderID, &d.CourierID, &statusStr, &d.RouteInstruction)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	d.Status = domain.DispatchStatus(statusStr)
	return &d, nil
}
