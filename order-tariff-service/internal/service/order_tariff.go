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
	"kopo":        1.25, // kopo macet parah: +25%
	"rancaekek":   1.15, // rancaekek banjir/macet: +15%
	"dayeuhkolot": 1.20, // dayeuhkolot banjir/macet: +20%
	"cibaduyut":   1.15, // cibaduyut macet: +15%
}

func (s *orderService) hitungTotalTarif(req domain.OrderRequest, dist float64) float64 {
	// 1. Hitung berat chargeable (mengambil max dari actual vs volumetric weight)
	volWeight := (req.Package.Length * req.Package.Width * req.Package.Height) / float64(PembagiVolumetrik)
	chargeableWeight := req.Package.ActualWeight
	if volWeight > chargeableWeight {
		chargeableWeight = volWeight
	}

	// Terapkan pembulatan berat khas ekspedisi (toleransi desimal 0.3 kg)
	chargeableWeight = bulatkanBerat(chargeableWeight)

	// Terapkan batas berat minimum untuk layanan kargo (CARGO)
	if strings.ToUpper(req.ServiceType) == "CARGO" {
		if chargeableWeight < 10.0 {
			chargeableWeight = 10.0 // Batas minimum kargo adalah 10 kg
		}
	}

	// 2. Ambil base rate per kg dari cache Redis (jika tersedia)
	baseRate, err := s.repo.GetPricingFromCache(req.ServiceType)
	if err != nil || baseRate <= 0 {
		baseRate = float64(HargaPerKg)
	}

	// 3. Hitung tarif dasar berdasarkan berat
	tarif := baseRate * chargeableWeight

	//tambahkan biaya jarak (misal Rp 100,- per km)
	tarif += dist * 100.0

	// tambahkan multiplier tipe layanan
	switch req.ServiceType {
	case "EXPRESS":
		tarif *= 1.5
	case "CARGO":
		tarif *= 0.8
	}

	// Tambahkan dynamic pricing untuk area macet (Congestion Surcharge)
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

// bulatkanBerat membulatkan berat paket sesuai aturan standar ekspedisi (misal: JNE / J&T)
// Jika kelebihan desimal > 0.3 kg, dibulatkan ke atas. Jika <= 0.3 kg, dibulatkan ke bawah.
func bulatkanBerat(berat float64) float64 {
	intPart := float64(int(berat))
	fracPart := berat - intPart
	if fracPart > 0.3 {
		return intPart + 1.0
	}
	if intPart == 0 {
		return 1.0 // minimal berat dihitung adalah 1 kg
	}
	return intPart
}
