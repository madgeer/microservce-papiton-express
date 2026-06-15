package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	
	"order-tariff-service/internal/domain"
)

/* OrderHandler menghubungkan request REST API dari client ke layer bisnis OrderService */
type OrderHandler struct {
	svc domain.OrderService
}

/* NewOrderHandler adalah constructor untuk inisialisasi handler */
func NewOrderHandler(svc domain.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

/* HandleCreateOrder memproses request POST /api/v1/orders untuk membuat order baru */
func (h *OrderHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Method tidak diizinkan, wajib POST",
		})
		return
	}

	var req domain.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Format payload JSON tidak valid",
		})
		return
	}

	// Panggil logika bisnis di layer service
	res, err := h.svc.CreateOrder(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Gagal membuat order: %v", err),
		})
		return
	}

	// Mengembalikan respons dengan status HTTP 201 Created
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

/* HandleGetOrders melayani GET /api/v1/orders (list) atau GET /api/v1/orders?awb=XXX (detail) */
func (h *OrderHandler) HandleGetOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Method tidak diizinkan, wajib GET",
		})
		return
	}

	awb := r.URL.Query().Get("awb")
	if awb != "" {
		res, err := h.svc.GetOrderByAWB(awb)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"message": fmt.Sprintf("Gagal mendapatkan order: %v", err),
			})
			return
		}
		if res == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Order tidak ditemukan",
			})
			return
		}
		json.NewEncoder(w).Encode(res)
		return
	}

	res, err := h.svc.GetAllOrders()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Gagal mendapatkan daftar order: %v", err),
		})
		return
	}
	if res == nil {
		res = []domain.OrderResponse{}
	}
	json.NewEncoder(w).Encode(res)
}

/* HandleCalculateTariff melayani POST /api/v1/tariff/calculate untuk simulasi cek ongkir */
func (h *OrderHandler) HandleCalculateTariff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Method tidak diizinkan, wajib POST",
		})
		return
	}

	var req domain.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Format payload JSON tidak valid",
		})
		return
	}

	res, err := h.svc.CalculateTariff(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Gagal menghitung tarif: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(res)
}
