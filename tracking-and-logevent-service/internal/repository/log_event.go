package repository

import "github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"

// LogEventRepository menangani operasi penulisan log ke database NoSQL (Write-Heavy).
type LogEventRepository interface {
	InsertLog(log model.TrackingLog) error
	FindAllLogs() ([]model.TrackingLog, error)
}