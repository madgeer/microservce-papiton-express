package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

/* TestGenerateLoadingInstruction_Unit untuk testing generate loading instruction */
func TestGenerateLoadingInstruction_Unit(t *testing.T) {
	svc := NewOperationalService()
	_, err := svc.GenerateLoadingInstruction("MNF-123")
	assert.ErrorIs(t, err, ErrNotImplemented)
}

/* TestGetCurrentWarehouseStock_Unit untuk testing get current warehouse stock */
func TestGetCurrentWarehouseStock_Unit(t *testing.T) {
	svc := NewOperationalService()
	_, err := svc.GetCurrentWarehouseStock("WH-001")
	assert.ErrorIs(t, err, ErrNotImplemented)
}
