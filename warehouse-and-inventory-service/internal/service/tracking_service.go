package service

/* TrackingService menangani komunikasi dengan service Tracking eksternal */
type TrackingService struct{}

/* NewTrackingService adalah constructor untuk inisialisasi service */
func NewTrackingService() *TrackingService {
	return &TrackingService{}
}

/* NotifyTrackingService memberi tahu Tracking Service bahwa status berubah */
func (s *TrackingService) NotifyTrackingService(resi string, status string) error {
	// Dikosongkan karena belum diimplementasikan, mengembalikan ErrNotImplemented
	return ErrNotImplemented
}

/* FormatManifestData merangkum data manifest untuk dikirim ke sistem eksternal */
func (s *TrackingService) FormatManifestData(manifestID string) (string, error) {
	// Dikosongkan karena belum diimplementasikan, mengembalikan ErrNotImplemented
	return "", ErrNotImplemented
}
