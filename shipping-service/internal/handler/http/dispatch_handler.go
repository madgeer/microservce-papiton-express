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

func (h *DispatchHandler) RegisterCourier(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib POST"})
		return
	}

	var req domain.Courier
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	err = h.service.RegisterCourier(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *DispatchHandler) GetCouriers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib GET"})
		return
	}

	id := r.URL.Query().Get("id")
	if id != "" {
		courier, err := h.service.GetCourier(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}
		if courier == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "courier tidak ditemukan"})
			return
		}
		json.NewEncoder(w).Encode(courier)
		return
	}

	zone := r.URL.Query().Get("zone")
	if zone == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "parameter 'zone' atau 'id' diperlukan"})
		return
	}

	list, err := h.service.GetAvailableCouriers(r.Context(), zone)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}
	if list == nil {
		list = []*domain.Courier{}
	}
	json.NewEncoder(w).Encode(list)
}

func (h *DispatchHandler) UpdateCourierStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib PUT atau POST"})
		return
	}

	var req struct {
		CourierID string `json:"courier_id"`
		Status    string `json:"status"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	err = h.service.UpdateCourierStatus(r.Context(), req.CourierID, domain.CourierStatus(req.Status))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "status updated"})
}

func (h *DispatchHandler) UpdateCourierLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib PUT atau POST"})
		return
	}

	var req struct {
		CourierID string  `json:"courier_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	err = h.service.UpdateCourierGPS(r.Context(), req.CourierID, req.Latitude, req.Longitude)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "location updated"})
}

func (h *DispatchHandler) ConfirmPickUp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib POST"})
		return
	}

	var req struct {
		DispatchID string `json:"dispatch_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	err = h.service.ConfirmPickUp(r.Context(), req.DispatchID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "pickup confirmed"})
}