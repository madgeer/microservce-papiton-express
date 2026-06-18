package postgres

import (
	"database/sql"
	"order-tariff-service/internal/domain"
	"order-tariff-service/internal/repository/redis"
	"strings"

	gredis "github.com/redis/go-redis/v9"
)

// struct implementasi repository
type OrderRepositoryImpl struct {
	db  *sql.DB
	rdb *gredis.Client
}

// constructor
func NewOrderRepository(db *sql.DB, rdb *gredis.Client) *OrderRepositoryImpl {
	return &OrderRepositoryImpl{db: db, rdb: rdb}
}

func (r *OrderRepositoryImpl) SaveOrder(req domain.OrderRequest, res domain.OrderResponse) error {
	// Pastikan tabel orders sudah terbuat di PostgreSQL
	queryCreateTable := `
	CREATE TABLE IF NOT EXISTS orders (
		awb VARCHAR(50) PRIMARY KEY,
		sender_name VARCHAR(100) NOT NULL,
		sender_phone VARCHAR(50) NOT NULL,
		sender_email VARCHAR(100) NOT NULL,
		sender_address TEXT NOT NULL,
		sender_city VARCHAR(50) NOT NULL,
		sender_lat DOUBLE PRECISION NOT NULL,
		sender_lng DOUBLE PRECISION NOT NULL,
		
		recipient_name VARCHAR(100) NOT NULL,
		recipient_phone VARCHAR(50) NOT NULL,
		recipient_email VARCHAR(100) NOT NULL,
		recipient_address TEXT NOT NULL,
		recipient_city VARCHAR(50) NOT NULL,
		recipient_lat DOUBLE PRECISION NOT NULL,
		recipient_lng DOUBLE PRECISION NOT NULL,
		
		package_length DOUBLE PRECISION NOT NULL,
		package_width DOUBLE PRECISION NOT NULL,
		package_height DOUBLE PRECISION NOT NULL,
		package_weight DOUBLE PRECISION NOT NULL,
		volumetric_weight DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		
		service_type VARCHAR(20) NOT NULL,
		has_insurance BOOLEAN NOT NULL DEFAULT FALSE,
		has_packing BOOLEAN NOT NULL DEFAULT FALSE,
		
		tarif_total DOUBLE PRECISION NOT NULL,
		distance DOUBLE PRECISION NOT NULL,
		eta VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := r.db.Exec(queryCreateTable)
	if err != nil {
		return err
	}

	queryInsert := `
	INSERT INTO orders (
		awb, sender_name, sender_phone, sender_email, sender_address, sender_city, sender_lat, sender_lng,
		recipient_name, recipient_phone, recipient_email, recipient_address, recipient_city, recipient_lat, recipient_lng,
		package_length, package_width, package_height, package_weight, volumetric_weight,
		service_type, has_insurance, has_packing,
		tarif_total, distance, eta, status
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15,
		$16, $17, $18, $19, $20,
		$21, $22, $23,
		$24, $25, $26, $27
	)`

	_, err = r.db.Exec(queryInsert,
		res.AWB, req.Sender.Name, req.Sender.Phone, req.Sender.Email, req.Sender.FullAddress, req.Sender.City, req.Sender.Coordinate.Latitude, req.Sender.Coordinate.Longitude,
		req.Recipient.Name, req.Recipient.Phone, req.Recipient.Email, req.Recipient.FullAddress, req.Recipient.City, req.Recipient.Coordinate.Latitude, req.Recipient.Coordinate.Longitude,
		req.Package.Length, req.Package.Width, req.Package.Height, req.Package.ActualWeight, req.Package.VolumetricWeight,
		req.ServiceType, req.HasInsurance, req.HasPacking,
		res.TarifTotal, res.Distance, res.ETA, res.Status,
	)
	return err
}

func (r *OrderRepositoryImpl) GetDistance(origin, dest domain.Koordinat) (float64, error) {
	var distanceKM float64

	// Query menggunakan ST_Distance dengan tipe data geography (spheroid WGS84)
	// ST_Distance mengembalikan satuan meter, sehingga dibagi 1000.0 agar menjadi KM
	query := `
		SELECT ST_Distance(
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
		) / 1000.0 AS distance_km;
	`

	// PostGIS ST_MakePoint menerima parameter: Longitude dahulu baru Latitude
	err := r.db.QueryRow(query, origin.Longitude, origin.Latitude, dest.Longitude, dest.Latitude).Scan(&distanceKM)
	if err != nil {
		return 0.0, err
	}

	return distanceKM, nil
}

func (r *OrderRepositoryImpl) GetPricingFromCache(key string) (float64, error) {
	return redis.GetPricing(r.rdb, key)
}

func (r *OrderRepositoryImpl) GetCityCode(cityName string) (string, error) {
	cityCodes := map[string]string{
		"BANDUNG":    "BDG",
		"JAKARTA":    "JKT",
		"SURABAYA":   "SUB",
		"SEMARANG":   "SMG",
		"MEDAN":      "MES",
		"MAKASSAR":   "UPG",
		"YOGYAKARTA": "YRK",
		"DENPASAR":   "DPS",
	}

	upperCity := strings.ToUpper(cityName)
	if code, found := cityCodes[upperCity]; found {
		return code, nil
	}

	if len(upperCity) >= 3 {
		return upperCity[:3], nil
	}
	return "BDG", nil
}

func (r *OrderRepositoryImpl) FindAll() ([]domain.OrderResponse, error) {
	querySelect := `SELECT awb, tarif_total, distance, eta, status, created_at FROM orders ORDER BY created_at DESC`
	rows, err := r.db.Query(querySelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.OrderResponse
	for rows.Next() {
		var o domain.OrderResponse
		err := rows.Scan(&o.AWB, &o.TarifTotal, &o.Distance, &o.ETA, &o.Status, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (r *OrderRepositoryImpl) FindByAWB(awb string) (*domain.OrderResponse, error) {
	querySelect := `SELECT awb, tarif_total, distance, eta, status, created_at FROM orders WHERE awb = $1`
	var o domain.OrderResponse
	err := r.db.QueryRow(querySelect, awb).Scan(&o.AWB, &o.TarifTotal, &o.Distance, &o.ETA, &o.Status, &o.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}
