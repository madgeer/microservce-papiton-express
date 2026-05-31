package service

import (
	"order-tariff-service/internal/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHitungETA(t *testing.T) {
	svc := &orderService{} // Inisialisasi struct langsung

	eta1 := svc.hitungETA(domain.ServiceTypeExpress, 50.0)
	assert.NotEmpty(t, eta1)
}
