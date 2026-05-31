package service

import "warehouse-inventory-service/internal/repository"

/*
* SortingService merepresentasikan domain logic untuk penyortiran paket.
* Berfungsi menentukan paket mana masuk ke keranjang/jalur (lane) tujuan yang mana.
*/
type SortingService struct {
	repo repository.SortingRepository
}

/* NewSortingService adalah constructor untuk inisialisasi service */
func NewSortingService(r repository.SortingRepository) *SortingService {
	return &SortingService{
		repo: r,
	}
}

/* AssignPackageToLane mengalokasikan paket ke jalur keberangkatan tertentu */
func (s *SortingService) AssignPackageToLane(resi string, laneID string) error {
	return s.repo.AssignToLane(resi, laneID)
}

/* GetPackagesInLane melihat daftar resi apa saja yang sudah ada di sebuah jalur */
func (s *SortingService) GetPackagesInLane(laneID string) ([]string, error) {
	return s.repo.GetPackagesInLane(laneID)
}
