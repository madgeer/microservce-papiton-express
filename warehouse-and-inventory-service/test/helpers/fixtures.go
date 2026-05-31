package helpers

import "fmt"

/*
* File fixtures.go berisi kumpulan data tiruan (dummy data) terpusat.
* Tujuannya agar tidak perlu men-hardcode string secara berulang
* di berbagai file test (seperti "BDO240430120000X1Y2").
*/

const (
	// DUMMY RESI
	DummyResiRegular = "BDO240430120000X1Y2"
	DummyResiExpress = "CGK240430120000EXP1"

	// === DUMMY WAREHOUSE / HUB ===
	DummyWarehouseOrigin = "WH-UPI"
	DummyWarehouseDest   = "WH-DESTINATION"

	// === DUMMY MANIFEST & FLEET ===
	DummyManifestID = "MNF-12345-ABC"
	DummyTruckID    = "TRK-B6990XYZ"
	DummyDriverName = "Sutejo"

	// === DUMMY STATUSES ===
	StatusAtHub    = "AT_HUB"
	StatusCreated  = "CREATED"
	StatusFinalized = "FINALIZED"
	StatusDeparted = "DEPARTED"
	StatusArrived  = "ARRIVED"
)

/*
* GenerateMultipleResi adalah fungsi pembantu jika sebuah functional test 
* membutuhkan banyak resi sekaligus (misal menguji kapasitas truk).
*/
func GenerateMultipleResi(count int) []string {
	var resiList []string
	for i := 1; i <= count; i++ {
		resiList = append(resiList, fmt.Sprintf("BDO-TEST-RESI-%03d", i))
	}
	return resiList
}
