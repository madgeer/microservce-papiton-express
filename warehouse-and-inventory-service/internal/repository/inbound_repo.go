package repository

/* inventory repository interface berisi fungsi-fungsi yang akan digunakan untuk berinteraksi dengan database */
type InboundRepository interface {
	// UpdateStockStatus digunakan untuk mengupdate status stok paket
	UpdateStockStatus(resi string, warehouseID string, status string) error

	// GetItemByResi digunakan untuk mendapatkan item berdasarkan resi
	GetItemByResi(resi string) (string, error)
	
	// UpdatePackageMetadata digunakan untuk mengupdate metadata paket
	UpdatePackageMetadata(resi string, instructions []string) error
}