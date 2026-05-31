//go:build functional

package functional_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"order-tariff-service/internal/domain"
	"order-tariff-service/internal/repository/postgres"
	"order-tariff-service/internal/service"
)

/*
functional test untuk order & tariff service
menguji flow createorder
fail pada tahap ini karena logic belum selesai
*/

func TestCreateOrderFlow_Functional(t *testing.T) {
	//setupdatabase
	db, err := postgres.InitDB()
	if err != nil {
		t.Fatalf("DB connection error: %v", err)
	}
	defer db.Close()

	//init repo & service
	repo := postgres.NewOrderRepository(db)
	svc := service.NewOrderService(repo)

	//dummy request
	req := domain.OrderRequest{
		Sender: domain.Customer{
			Name:        "Shidqi",
			Phone:       "08123456789",
			Email:       "shidqi@test.com",
			FullAddress: "Rancaekek Permai",
			City:        "Bandung",
			Coordinate: domain.Koordinat{
				Latitude:  -6.9175,
				Longitude: 107.6191,
			},
		},
		Recipient: domain.Customer{
			Name:        "Asep",
			Phone:       "0812937123712",
			Email:       "asep@test.com",
			FullAddress: "Jakarta Timur",
			City:        "Jakarta",
			Coordinate: domain.Koordinat{
				Latitude:  -6.1751,
				Longitude: 106.8272,
			},
		},
		Package: domain.Paket{
			Length:       10,
			Width:        10,
			Height:       10,
			ActualWeight: 2.5,
		},
		ServiceType:  domain.ServiceTypeRegular,
		HasInsurance: false,
		HasPacking:   false,
	}

	//excurte function
	res, err := svc.CreateOrder(req)

	//assertion
	assert.NoError(t, err)

	assert.NotEmpty(t, res.AWB, "AWB harud digenerate")
	assert.Greater(t, res.TarifTotal, 0.0, "Tarif harus lebih dari 0")
	assert.NotEmpty(t, res.ETA, "ETA harus terisi")
}
