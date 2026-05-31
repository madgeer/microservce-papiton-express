package repository

import "github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"

// TrackingRepository menangani operasi pembacaan histori pelacakan (Read-Heavy).
type TrackingRepository interface {
	GetResiHistory(resiID string) (*model.TrackingHistory, error)
}