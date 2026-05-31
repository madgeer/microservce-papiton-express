package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

/* TestNotifyTrackingService_Unit untuk testing notify tracking service */
func TestNotifyTrackingService_Unit(t *testing.T) {
	svc := NewTrackingService()
	err := svc.NotifyTrackingService("RESI-001", "IN_TRANSIT")
	assert.ErrorIs(t, err, ErrNotImplemented)
}

/* TestFormatManifestData_Unit untuk testing format manifest data */
func TestFormatManifestData_Unit(t *testing.T) {
	svc := NewTrackingService()
	_, err := svc.FormatManifestData("MNF-123")
	assert.ErrorIs(t, err, ErrNotImplemented)
}
