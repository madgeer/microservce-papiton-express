package main

import (
	"log"
	"net/http"

	handlerHttp "github.com/madgeer/papiton-express/shipping-service/internal/handler/http"
)

func main() {
	// Pada tahap ini kita menginjeksi nil ke handler 
	// karena kita masih dalam Red Phase (belum ada service utuh)
	dispatchHandler := handlerHttp.NewDispatchHandler(nil)

	// Daftarkan endpoint REST API
	http.HandleFunc("/dispatch", dispatchHandler.AutoDispatch)

	// Start server di port 8080
	log.Println("Shipping & Dispatch Service berjalan di port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}