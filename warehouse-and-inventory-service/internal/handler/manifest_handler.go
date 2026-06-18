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
* ManifestHandler menghubungkan request HTTP dari client ke logika bisnis ManifestService.
*/
type ManifestHandler struct {
	svc *service.ManifestService
}

// NewManifestHandler inisialisasi handler
func NewManifestHandler(svc *service.ManifestService) *ManifestHandler {
	return &ManifestHandler{
		svc: svc,
	}
}

/*
* HandleCreateManifest menghandle request untuk membuat manifest baru
*/
func (h *ManifestHandler) HandleCreateManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
		return
	}

	var req model.CreateManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"Format JSON tidak valid"}`, http.StatusBadRequest)
		return
	}

	manifestID, err := h.svc.CreateNewManifest(req.TruckID, req.DriverName)
	if err != nil {
		http.Error(w, `{"message":"Gagal membuat manifest"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.ManifestResponse{
		Status:     http.StatusOK,
		Message:    "Manifest berhasil dibuat",
		ManifestID: manifestID,
	})
}

/*
* HandleAddToManifest menghandle request untuk memasukkan paket ke dalam manifest
*/
func (h *ManifestHandler) HandleAddToManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
		return
	}

	var req model.AddToManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"Format JSON tidak valid"}`, http.StatusBadRequest)
		return
	}

	err := h.svc.AddToManifest(req.ManifestID, req.Resi)
	if err != nil {
		log.Printf("ERROR HandleAddToManifest: %v\n", err)
		http.Error(w, fmt.Sprintf(`{"message":"Gagal menambah paket ke manifest: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.ManifestResponse{
		Status:     http.StatusOK,
		Message:    "Paket berhasil dimasukkan ke manifest",
		ManifestID: req.ManifestID,
	})
}

/*
* HandleUpdateManifest menghandle request untuk mengubah status manifest
*/
func (h *ManifestHandler) HandleUpdateManifest(w http.ResponseWriter, r *http.Request) {
	// Mengatur header HTTP untuk respons berformat JSON
	w.Header().Set("Content-Type", "application/json")
	
	// Melakukan validasi metode HTTP (diwajibkan POST)
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
		return
	}

	// Melakukan dekode JSON Payload dari Body Request ke dalam struct UpdateManifestRequest
	var req model.UpdateManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"Format JSON tidak valid"}`, http.StatusBadRequest)
		return
	}

	// Pada skenario nyata, URL path (misalnya /finalize, /depart, /receive)
	// menentukan fungsi service mana yang akan dipanggil. 
	// Karena ini adalah template umum, simulasi pemanggilan Depart sebagai contoh
	err := h.svc.DepartManifest(req.ManifestID)
	if err != nil {
		http.Error(w, `{"message":"Gagal update manifest"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.ManifestResponse{
		Status:     http.StatusOK,
		Message:    "Status manifest berhasil di-update",
		ManifestID: req.ManifestID,
	})
}
