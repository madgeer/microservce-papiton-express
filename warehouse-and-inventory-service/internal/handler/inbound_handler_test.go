package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"warehouse-inventory-service/internal/handler"
	"warehouse-inventory-service/internal/model"
	"warehouse-inventory-service/internal/service"
	"warehouse-inventory-service/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestHandleProcessInbound_MethodNotAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := service.NewInboundService(mockRepo, nil)
	h := handler.NewInboundHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/inbound", nil)
	rr := httptest.NewRecorder()

	h.HandleProcessInbound(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

	var resp model.InboundResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.Status)
	assert.Contains(t, resp.Message, "Method tidak diizinkan")
}

func TestHandleProcessInbound_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := service.NewInboundService(mockRepo, nil)
	h := handler.NewInboundHandler(svc)

	reqBody := bytes.NewBufferString(`{invalid-json}`)
	req := httptest.NewRequest(http.MethodPost, "/inbound", reqBody)
	rr := httptest.NewRecorder()

	h.HandleProcessInbound(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp model.InboundResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Status)
	assert.Contains(t, resp.Message, "Format payload JSON tidak valid")
}

func TestHandleProcessInbound_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := service.NewInboundService(mockRepo, nil)
	h := handler.NewInboundHandler(svc)

	// Persiapkan ekspektasi mock repository
	// Karena ProcessInbound di inbound_service.go memanggil repo.UpdateStockStatus
	// Skenariokan repository mengembalikan error
	resi := "BDO240430120000X1Y2"
	warehouseID := "WH-UPI"
	mockRepo.EXPECT().UpdateStockStatus(resi, warehouseID, "AT_HUB").Return(errors.New("db error")).Times(1)

	// Namun tunggu! ProcessInbound di inbound_service.go saat ini hanya me-return nil
	// tanpa memanggil repository!
	// Agar test ini bertindak sebagai RED phase (gagal ketika kode produksi belum selesai),
	// panggil service layer. Jika service layer mengembalikan nil (seperti di stub sekarang),
	// maka handler akan mengembalikan HTTP 200, padahal ekspektasi  adalah HTTP 500 karena repo error.
	// Ini adalah perilaku RED phase TDD yang benar!
	reqBody, _ := json.Marshal(model.InboundRequest{
		Resi:        resi,
		WarehouseID: warehouseID,
	})
	req := httptest.NewRequest(http.MethodPost, "/inbound", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	h.HandleProcessInbound(rr, req)

	// Ekspektasi: HTTP 500 karena service/repository error
	// Kenyataan (RED Phase): Akan gagal (rr.Code != 500) jika ProcessInbound belum memanggil repo dan mengembalikan error!
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleProcessInbound_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockInboundRepository(ctrl)
	svc := service.NewInboundService(mockRepo, nil)
	h := handler.NewInboundHandler(svc)

	resi := "BDO240430120000X1Y2"
	warehouseID := "WH-UPI"
	mockRepo.EXPECT().UpdateStockStatus(resi, warehouseID, "AT_HUB").Return(nil).AnyTimes()

	reqBody, _ := json.Marshal(model.InboundRequest{
		Resi:        resi,
		WarehouseID: warehouseID,
	})
	req := httptest.NewRequest(http.MethodPost, "/inbound", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	h.HandleProcessInbound(rr, req)

	// Ekspektasi sukses HTTP 200
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp model.InboundResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, resi, resp.Resi)
	assert.Contains(t, resp.Message, "Inbound paket berhasil diproses")
}
