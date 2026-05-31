package model

/*
* CreateManifestRequest merepresentasikan payload JSON saat client 
* meminta pembuatan manifest/truk baru.
*/
type CreateManifestRequest struct {
	TruckID    string `json:"truck_id" binding:"required"`
	DriverName string `json:"driver_name" binding:"required"`
}

/*
* AddToManifestRequest merepresentasikan payload JSON saat client
* memasukkan sebuah paket (resi) ke dalam manifest truk.
*/
type AddToManifestRequest struct {
	ManifestID string `json:"manifest_id" binding:"required"`
	Resi       string `json:"resi" binding:"required"`
}

/*
* UpdateManifestRequest bisa digunakan sebagai kerangka JSON untuk aksi:
* Finalize, Depart, dan Receive. 
* WarehouseID digunakan khusus saat truk telah tiba (Receive) di gudang tujuan.
*/
type UpdateManifestRequest struct {
	ManifestID  string `json:"manifest_id" binding:"required"`
	WarehouseID string `json:"warehouse_id,omitempty"` // Opsional, hanya untuk Receive
}

/*
* ManifestResponse merepresentasikan bentuk standar JSON balasan (response)
* yang akan dikirimkan kembali ke klien setelah memanggil API Manifest.
*/
type ManifestResponse struct {
	Status     int    `json:"status"`
	Message    string `json:"message"`
	ManifestID string `json:"manifest_id,omitempty"`
}
