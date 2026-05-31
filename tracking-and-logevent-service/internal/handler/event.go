package handler

import (
	"encoding/json"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/service"
)

type EventConsumerHandler struct {
	svc *service.LogEventService
}

func NewEventConsumerHandler(svc *service.LogEventService) *EventConsumerHandler {
	return &EventConsumerHandler{svc: svc}
}

func (h *EventConsumerHandler) ConsumeLogEvent(message []byte) {
	var log model.TrackingLog
	if err := json.Unmarshal(message, &log); err != nil {
		return // Silently fail atau log error untuk background process
	}
	
	_ = h.svc.ProcessLog(log)
}
