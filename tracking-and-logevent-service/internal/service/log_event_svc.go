package service

import (
	"context"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/repository/kafka"
)

type LogEventService struct {
	repo      repository.LogEventRepository
	publisher kafka.TrackingEventPublisher
}

func NewLogEventService(repo repository.LogEventRepository, publisher kafka.TrackingEventPublisher) *LogEventService {
	return &LogEventService{repo: repo, publisher: publisher}
}

func (s *LogEventService) ProcessLog(log model.TrackingLog) error {
	// Validasi sederhana sebelum simpan
	if log.ResiID == "" {
		return repository.ErrInvalidData
	}
	err := s.repo.InsertLog(log)
	if err != nil {
		return err
	}

	// Kirim event asinkron ke Kafka untuk pembaruan status pengiriman di DWH
	if s.publisher != nil {
		go func() {
			_ = s.publisher.PublishTrackingLog(context.Background(), log)
		}()
	}
	return nil
}

func (s *LogEventService) GetAllLogs() ([]model.TrackingLog, error) {
	return s.repo.FindAllLogs()
}
