package service

import (
	"math/rand"
	"time"
)

func (s *orderService) GenerateAWB(cityName string) string {
	code, err := s.repo.GetCityCode(cityName)
	if err != nil {
		code = "BDG"
	}
	now := time.Now().Format("060102150405") // YYMMDDHHMMSS
	
	// 4 char random
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	randomPart := ""
	for i := 0; i < 4; i++ {
		randomPart += string(chars[rand.Intn(len(chars))])
	}
	return code + now + randomPart
}
