package service

import (
	"order-tariff-service/internal/domain"
	"strings"
)

const (
	PembagiVolumetrik = 6000
	HargaPerKg        = 5000
)

var MacetZones = map[string]float64{
	"kopo":        1.25, // Kopo macet parah: +25%
	"rancaekek":   1.15, // Rancaekek banjir/macet: +15%
	"dayeuhkolot": 1.20, // Dayeuhkolot banjir/macet: +20%
	"cibaduyut":   1.15, // Cibaduyut macet: +15%
}

func (s *orderService) hitungTotalTarif(req domain.OrderRequest, dist float64) float64 {
	// 1. Hitung berat chargeable (mengambil max dari actual vs volumetric weight)
	volWeight := (req.Package.Length * req.Package.Width * req.Package.Height) / float64(PembagiVolumetrik)
	chargeableWeight := req.Package.ActualWeight
	if volWeight > chargeableWeight {
		chargeableWeight = volWeight
	}
	if chargeableWeight <= 0 {
		chargeableWeight = 1.0 // minimal chargeable weight adalah 1 kg
	}

	// 2. Ambil base rate per kg dari cache Redis (jika tersedia)
	baseRate, err := s.repo.GetPricingFromCache(req.ServiceType)
	if err != nil || baseRate <= 0 {
		baseRate = float64(HargaPerKg)
	}

	// 3. Hitung tarif dasar berdasarkan berat
	tarif := baseRate * chargeableWeight

	// 4. Tambahkan biaya jarak (misal Rp 1.000,- per km)
	tarif += dist * 1000.0

	// 5. Tambahkan multiplier tipe layanan
	switch req.ServiceType {
	case "EXPRESS":
		tarif *= 1.5
	case "CARGO":
		tarif *= 0.8
	}

	// 5.1. Tambahkan dynamic pricing untuk area macet (Congestion Surcharge)
	addrLower := strings.ToLower(req.Recipient.FullAddress)
	cityLower := strings.ToLower(req.Recipient.City)
	for zone, multiplier := range MacetZones {
		if strings.Contains(addrLower, zone) || strings.Contains(cityLower, zone) {
			tarif *= multiplier
			break
		}
	}

	// 6. Tambahkan asuransi dan packing jika dipilih
	if req.HasInsurance {
		tarif += 10000.0
	}
	if req.HasPacking {
		tarif += 20000.0
	}

	return tarif
}
