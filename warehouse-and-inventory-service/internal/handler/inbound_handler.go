package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"warehouse-inventory-service/internal/model"
	"warehouse-inventory-service/internal/service"
)

/*
* InboundHandler bertanggung jawab menerima request HTTP dari client,
* melakukan validasi format (JSON), dan meneruskannya ke layer Service.
*/
type InboundHandler struct {
	svc *service.InboundService
}

/*
* NewInboundHandler adalah constructor untuk inisialisasi handler
*/
func NewInboundHandler(svc *service.InboundService) *InboundHandler {
	return &InboundHandler{
		svc: svc,
	}
}

/*
* HandleProcessInbound menangani endpoint untuk memproses paket baru masuk
*/
func (h *InboundHandler) HandleProcessInbound(w http.ResponseWriter, r *http.Request) {
	// Mengatur header HTTP untuk respons berformat JSON
	w.Header().Set("Content-Type", "application/json")

	// Melakukan validasi metode HTTP (diwajibkan POST)
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(model.InboundResponse{
			Status:  http.StatusMethodNotAllowed,
			Message: "Method tidak diizinkan, wajib POST",
		})
		return
	}

	// Melakukan dekode JSON Payload dari Body Request ke dalam struct InboundRequest
	var req model.InboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.InboundResponse{
			Status:  http.StatusBadRequest,
			Message: "Format payload JSON tidak valid",
		})
		return
	}

	// Meneruskan data ke layer Service untuk eksekusi logika bisnis
	err := h.svc.ProcessInbound(req.Resi, req.WarehouseID)
	if err != nil {
		// Menangani error yang dikembalikan oleh layer Service
		log.Printf("ERROR HandleProcessInbound: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(model.InboundResponse{
			Status:  http.StatusInternalServerError,
			Message: fmt.Sprintf("Gagal memproses inbound: %v", err),
		})
		return
	}

	// Mengirimkan respons HTTP 200 (OK) apabila proses berhasil
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.InboundResponse{
		Status:  http.StatusOK,
		Message: "Inbound paket berhasil diproses",
		Resi:    req.Resi,
	})
}
