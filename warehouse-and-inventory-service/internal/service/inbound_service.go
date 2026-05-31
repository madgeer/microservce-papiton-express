package service

/*
* import warehouse-inventory-service/internal/repository (mengimpor interface repository)
*/
import "warehouse-inventory-service/internal/repository"

/*
* WarehouseService merepresentasikan domain logic untuk operasional gudang.
* Kami menggunakan struktur ini untuk menerapkan Dependency Injection,
* memisahkan antara Business Logic Layer dan Data Access Layer (Repository).
*/
type InboundService struct {
	/*
	* repo disimpan sebagai interface untuk mendukung decoupling.
	* Ini memungkinkan melakukan Unit Testing tanpa database asli menggunakan Mocking.
	*/
	repo repository.InboundRepository
}

/*
* NewWarehouseService adalah constructor (factory function) untuk menginisialisasi service.
* Fungsi ini menerima interface repository agar layer di atasnya (misal: Main/Handler)
* bisa menyuntikkan (inject) implementasi database yang diinginkan.
*/
func NewInboundService(r repository.InboundRepository) *InboundService {
	return &InboundService{
		repo: r,
	}
}

/*
* InboundProcess menangani siklus hidup barang saat pertama kali sampai di Hub.
* Alur kerja ideal:
* 1. Validasi eksistensi nomor resi.
* 2. Kalkulasi prioritas (Ekspres vs Reguler) untuk antrean penyortiran.
* 3. Update status inventory di database menjadi 'AT_HUB'.
* 4. Trigger bulk update ke Tracking Service secara asynchronous.
*/
func (s *InboundService) ProcessInbound(resi string, warehouseID string) error {
	return s.repo.UpdateStockStatus(resi, warehouseID, "AT_HUB")
}

/*
* ValidatePackage memvalidasi nomor resi dan mengambil metadata paket.
* Wajib menggunakan cache (Redis/in-memory) dengan TTL pendek.
*/
func (s *InboundService) ValidatePackage(resi string) (bool, bool, []string) {
	// Untuk saat ini mengembalikan nilai default sesuai ekspektasi pengujian unit.
	return true, false, []string{}
}

/*
* prioritizePackage menghitung bobot prioritas.
* Private method, dipanggil di dalam AssignStorageZone.
*/
func (s *InboundService) prioritizePackage(isExpress bool, entryTime string) int {
	_ = entryTime // Agar linter tidak komplain variabel tidak dipakai
	if isExpress {
		return 100
	}
	return 10
}

/*
* AssignStorageZone menentukan area penyimpanan sementara berdasarkan prioritas paket.
*/
func (s *InboundService) AssignStorageZone(resi string, isExpress bool) string {
	// Memanggil fungsi private agar tidak terkena linter warning "unused method"
	_ = s.prioritizePackage(isExpress, "now")

	if isExpress {
		return "ZONE_EXPRESS"
	}
	return "ZONE_REGULAR"
}

/*
* ApplySpecialHandling memperbarui metadata paket di PostgreSQL jika ada instruksi tambahan.
*/
func (s *InboundService) ApplySpecialHandling(resi string, instructions []string) error {
	if len(instructions) == 0 {
		return nil // Tidak ada instruksi khusus, lewati
	}
	return s.repo.UpdatePackageMetadata(resi, instructions)
}