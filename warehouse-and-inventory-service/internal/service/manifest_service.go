package service

import "warehouse-inventory-service/internal/repository"

/*
* ManifestService merepresentasikan domain logic untuk pengelolaan 
* manifest (pengelompokan paket ke dalam satu truk/pengiriman).
*/
type ManifestService struct {
	repo repository.ManifestRepository 
}

/* NewManifestService adalah constructor untuk inisialisasi service */
func NewManifestService(r repository.ManifestRepository) *ManifestService {
	return &ManifestService{
		repo: r,
	}
}

/* CreateNewManifest membuat dokumen manifest baru untuk truk tertentu */
func (s *ManifestService) CreateNewManifest(truckID string, driverName string) (string, error) {
	return s.repo.CreateManifest(truckID, driverName)
}

/* AddToManifest memasukkan paket (resi) ke dalam manifest yang sudah dibuat */
func (s *ManifestService) AddToManifest(manifestID string, resi string) error {
	return s.repo.AddPackageToManifest(manifestID, resi)
}

/* FinalizeManifest mengunci manifest sehingga tidak bisa ditambah paket lagi (siap jalan) */
func (s *ManifestService) FinalizeManifest(manifestID string) error {
	return s.repo.UpdateManifestStatus(manifestID, "FINALIZED")
}

/* DepartManifest menandai bahwa truk/manifest tersebut sudah berangkat dari Hub */
func (s *ManifestService) DepartManifest(manifestID string) error {
	return s.repo.UpdateManifestStatus(manifestID, "DEPARTED")
}

/* ReceiveManifest menandai bahwa manifest telah sampai di Hub/Warehouse tujuan */
func (s *ManifestService) ReceiveManifest(manifestID string, warehouseID string) error {
	return s.repo.UpdateManifestStatus(manifestID, "ARRIVED")
}
