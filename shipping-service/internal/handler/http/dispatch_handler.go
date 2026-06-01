package http

import (
	"encoding/json"
	"net/http"

	"github.com/madgeer/papiton-express/shipping-service/internal/domain"
)

type DispatchHandler struct {
	service domain.DispatchService
}

type autoDispatchRequest struct {
	OrderID    string `json:"order_id"`
	PickupZone string `json:"pickup_zone"`
}

func NewDispatchHandler(svc domain.DispatchService) *DispatchHandler {
	return &DispatchHandler{service: svc}
}

func (h *DispatchHandler) AutoDispatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib POST"})
		return
	}

	var req autoDispatchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	if req.OrderID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "order_id kosong"})
		return
	}

	if h.service == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "service tidak diinisialisasi"})
		return
	}

	dispatch, err := h.service.AutoDispatchPickUp(r.Context(), req.OrderID, req.PickupZone)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dispatch)
}