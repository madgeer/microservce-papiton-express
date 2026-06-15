package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
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

type LogEventAPIHandler struct {
	svc *service.LogEventService
}

func NewLogEventAPIHandler(svc *service.LogEventService) *LogEventAPIHandler {
	return &LogEventAPIHandler{svc: svc}
}

func (h *LogEventAPIHandler) ScanLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib POST"})
		return
	}

	var req model.TrackingLog
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	err = h.svc.ProcessLog(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *LogEventAPIHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib GET"})
		return
	}

	list, err := h.svc.GetAllLogs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(list)
}
