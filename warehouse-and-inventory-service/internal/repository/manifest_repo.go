package repository

/*
* ManifestRepository adalah interface yang berisi fungsi-fungsi 
* untuk berinteraksi dengan tabel Manifest di database.
*/
type ManifestRepository interface {
	// Membuat record manifest baru di DB dan mengembalikan ID-nya
	CreateManifest(truckID string, driverName string) (string, error)
	
	// Menambahkan relasi paket (resi) ke dalam manifest tertentu
	AddPackageToManifest(manifestID string, resi string) error
	
	// Mengubah status manifest (misalnya dari "CREATED" -> "DEPARTED" -> "ARRIVED")
	UpdateManifestStatus(manifestID string, status string) error
	
	// Mengecek status dan keberadaan manifest di DB
	GetManifestStatus(manifestID string) (string, error)
}
