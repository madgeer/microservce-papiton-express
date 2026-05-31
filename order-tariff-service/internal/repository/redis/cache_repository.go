package redis

type RedisRepo struct {
	//akan menambahkan redis client
}

func (r *RedisRepo) GetPricingFromCache(key string) (float64, error) {
	return 0.0, nil
}
