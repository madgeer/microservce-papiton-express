package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

/* TestUpdatePackageStatus_Unit untuk testing update package status */
func TestUpdatePackageStatus_Unit(t *testing.T) {
	svc := NewStatusService()
	err := svc.UpdatePackageStatus("RESI-001", "ARRIVED")
	assert.ErrorIs(t, err, ErrNotImplemented)
}
