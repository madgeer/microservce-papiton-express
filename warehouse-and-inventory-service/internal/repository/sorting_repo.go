package repository

/*
* SortingRepository adalah interface untuk interaksi database
* terkait fitur penyortiran paket ke dalam jalur (lane) truk.
*/
type SortingRepository interface {
	// AssignToLane digunakan untuk menugaskan paket ke jalur truk
	AssignToLane(resi string, laneID string) error
	
	// GetPackagesInLane digunakan untuk mendapatkan paket dalam jalur truk
	GetPackagesInLane(laneID string) ([]string, error)
}
