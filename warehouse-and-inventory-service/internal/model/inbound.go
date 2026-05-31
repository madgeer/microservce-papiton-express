package model

/*
* InboundRequest merepresentasikan payload JSON yang akan diterima API 
* saat ada paket baru yang masuk (inbound) ke warehouse.
*/
type InboundRequest struct {
	Resi         string   `json:"resi" binding:"required"`
	WarehouseID  string   `json:"warehouse_id" binding:"required"`
	Instructions []string `json:"instructions,omitempty"`
}

/*
* InboundResponse merepresentasikan payload JSON yang akan dikembalikan oleh API
* sebagai respons atas operasi inbound.
*/
type InboundResponse struct {
	Status      int    `json:"status"`
	Message     string `json:"message"`
	Resi        string `json:"resi,omitempty"`
	StorageZone string `json:"storage_zone,omitempty"`
}
