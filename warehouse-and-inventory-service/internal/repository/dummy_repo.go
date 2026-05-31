package repository

import "errors"

/*
* DummyInboundRepo adalah implementasi sementara (stub) dari InboundRepository.
* Dibuat HANYA agar server HTTP bisa dijalankan (compile) pada Tugas Tahap 2,
* karena implementasi database aslinya (PostgreSQL) belum dikerjakan.
*/
type DummyInboundRepo struct{}

/* NewDummyInboundRepo membuat instance baru dari Dummy Repo */
func NewDummyInboundRepo() *DummyInboundRepo {
	return &DummyInboundRepo{}
}

/* UpdateStockStatus mengupdate status stok paket */
func (r *DummyInboundRepo) UpdateStockStatus(resi string, warehouseID string, status string) error {
	return nil
}

/* GetItemByResi mengembalikan error seolah-olah data belum ada/belum tersambung DB */
func (r *DummyInboundRepo) GetItemByResi(resi string) (string, error) {
	return "", errors.New("database belum diimplementasikan")
}

/* UpdatePackageMetadata mengupdate metadata paket */
func (r *DummyInboundRepo) UpdatePackageMetadata(resi string, instructions []string) error {
	return nil
}
