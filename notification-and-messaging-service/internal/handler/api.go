package handler

import (
	"encoding/json"
	"net/http"
	"context"

	"papiton/notification-service/internal/model"
	"papiton/notification-service/internal/repository"
)

type NotificationLogReader interface {
	GetLogs(ctx context.Context) ([]repository.NotificationLog, error)
}

type MessageDispatcher interface {
	Dispatch(ctx context.Context, msg model.NotificationMessage) error
}

type NotificationAPIHandler struct {
	dispatcher MessageDispatcher
	repo       NotificationLogReader
}

func NewNotificationAPIHandler(d MessageDispatcher, r NotificationLogReader) *NotificationAPIHandler {
	return &NotificationAPIHandler{
		dispatcher: d,
		repo:       r,
	}
}

func (h *NotificationAPIHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib GET"})
		return
	}

	logs, err := h.repo.GetLogs(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(logs)
}

func (h *NotificationAPIHandler) SendDirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method tidak diizinkan, wajib POST"})
		return
	}

	var req struct {
		UserID  string `json:"user_id"`
		Channel string `json:"channel"` // email atau push
		Subject string `json:"subject"`
		Body    string `json:"body"`
		AWB     string `json:"awb"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid json"})
		return
	}

	msg := model.NotificationMessage{
		UserID:  req.UserID,
		Channel: model.Channel(req.Channel),
		Subject: req.Subject,
		Body:    req.Body,
		AWB:     req.AWB,
	}

	err = h.dispatcher.Dispatch(r.Context(), msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "notification dispatched successfully"})
}
