package postgres

import (
	"database/sql"
	"math"
	"order-tariff-service/internal/domain"
	"strings"
)

// struct implementasi repository
type OrderRepositoryImpl struct {
	db *sql.DB
}

// constructor
func NewOrderRepository(db *sql.DB) *OrderRepositoryImpl {
	return &OrderRepositoryImpl{db: db}
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
		package_length, package_width, package_height, package_weight,
		service_type, has_insurance, has_packing,
		tarif_total, distance, eta, status
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15,
		$16, $17, $18, $19,
		$20, $21, $22,
		$23, $24, $25, $26
	)`

	_, err = r.db.Exec(queryInsert,
		res.AWB, req.Sender.Name, req.Sender.Phone, req.Sender.Email, req.Sender.FullAddress, req.Sender.City, req.Sender.Coordinate.Latitude, req.Sender.Coordinate.Longitude,
		req.Recipient.Name, req.Recipient.Phone, req.Recipient.Email, req.Recipient.FullAddress, req.Recipient.City, req.Recipient.Coordinate.Latitude, req.Recipient.Coordinate.Longitude,
		req.Package.Length, req.Package.Width, req.Package.Height, req.Package.ActualWeight,
		req.ServiceType, req.HasInsurance, req.HasPacking,
		res.TarifTotal, res.Distance, res.ETA, res.Status,
	)
	return err
}

func (r *OrderRepositoryImpl) GetDistance(origin, dest domain.Koordinat) (float64, error) {
	// Formula Haversine untuk menghitung jarak antara dua titik koordinat GPS
	const R = 6371.0 // Radius bumi dalam kilometer

	lat1 := origin.Latitude * math.Pi / 180.0
	lng1 := origin.Longitude * math.Pi / 180.0
	lat2 := dest.Latitude * math.Pi / 180.0
	lng2 := dest.Longitude * math.Pi / 180.0

	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	return distance, nil
}

func (r *OrderRepositoryImpl) GetPricingFromCache(key string) (float64, error) {
	// Skenario fallback jika Redis belum tersambung
	return 0.0, nil
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
