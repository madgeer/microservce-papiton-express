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

	var req autoDispatchRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.OrderID == "" {
		http.Error(w, "order_id kosong", http.StatusBadRequest)
		return
	}

	http.Error(w, "not implemented", http.StatusNotImplemented)
}