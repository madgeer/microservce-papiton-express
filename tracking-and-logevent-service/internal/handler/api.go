package handler

import (
	"encoding/json"
	"net/http"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/service"
)

type TrackingAPIHandler struct {
	svc *service.TrackingService
}

func NewTrackingAPIHandler(svc *service.TrackingService) *TrackingAPIHandler {
	return &TrackingAPIHandler{svc: svc}
}

func (h *TrackingAPIHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	resiID := r.URL.Query().Get("resi_id")
	if resiID == "" {
		http.Error(w, "resi_id is required", http.StatusBadRequest)
		return
	}

	history, err := h.svc.GetHistory(resiID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}
