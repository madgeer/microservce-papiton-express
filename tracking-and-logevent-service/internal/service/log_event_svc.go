package service

import (
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
)

type LogEventService struct {
	repo repository.LogEventRepository
}

func NewLogEventService(repo repository.LogEventRepository) *LogEventService {
	return &LogEventService{repo: repo}
}

func (s *LogEventService) ProcessLog(log model.TrackingLog) error {
	// Validasi sederhana sebelum simpan
	if log.ResiID == "" {
		return repository.ErrInvalidData
	}
	return s.repo.InsertLog(log)
}

func (s *LogEventService) GetAllLogs() ([]model.TrackingLog, error) {
	return s.repo.FindAllLogs()
}
