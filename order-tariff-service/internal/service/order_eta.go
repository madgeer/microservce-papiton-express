package service

func (s *orderService) hitungETA(serviceType string, dist float64) string {
	switch serviceType {
	case "EXPRESS":
		if dist <= 100.0 {
			return "1 Hari"
		}
		return "2 Hari"
	case "REGULAR":
		if dist <= 100.0 {
			return "2-3 Hari"
		}
		return "3-5 Hari"
	default:
		return "5-7 Hari"
	}
}
