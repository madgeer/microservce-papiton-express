package http_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	handlerHttp "github.com/madgeer/papiton-express/shipping-service/internal/handler/http"
)

// Mock sederhana (karena kita di Red Phase, kita bisa pakai mock manual atau gomock)
// Untuk tes validasi JSON, kita bisa passing nil ke service.
func TestAutoDispatch_EmptyOrderID_Returns400(t *testing.T) {
	// Arrange: Setup Handler dengan service nil (karena harusnya gagal sebelum panggil service)
	h := handlerHttp.NewDispatchHandler(nil)

	// Membuat dummy request JSON dengan order_id yang kosong
	reqBody := bytes.NewBufferString(`{"order_id": "", "pickup_zone": "Bandung"}`)
	
	// Membuat simulasi HTTP Request (POST /dispatch)
	req := httptest.NewRequest(http.MethodPost, "/dispatch", reqBody)
	req.Header.Set("Content-Type", "application/json")

	// ResponseRecorder berfungsi seperti Postman yang menangkap balasan dari API
	rr := httptest.NewRecorder()

	// Act: Eksekusi Handler
	h.AutoDispatch(rr, req)

	// Assert: Kita mengekspektasikan Handler menolak karena order_id kosong (Status 400)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Ekspektasi status %d (Bad Request), tapi dapat %d", http.StatusBadRequest, rr.Code)
	}
}