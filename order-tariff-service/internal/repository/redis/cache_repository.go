package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func InitRedis() (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("[Redis] Menghubungkan ke Redis di %s...", addr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke Redis: %v", err)
	}

	log.Println("[Redis] Sukses! Terhubung ke Redis")
	return rdb, nil
}

func GetPricing(rdb *redis.Client, key string) (float64, error) {
	if rdb == nil {
		return 0, fmt.Errorf("redis client tidak diinisialisasi")
	}

	ctx := context.Background()
	val, err := rdb.Get(ctx, "pricing:"+key).Result()
	if err == redis.Nil {
		// Cache miss - set default rate
		var defaultRate float64
		switch key {
		case "EXPRESS":
			defaultRate = 7500.0
		case "CARGO":
			defaultRate = 3500.0
		default:
			defaultRate = 5000.0 // REGULAR or others
		}

		log.Printf("[Redis] Cache miss untuk key '%s'. Menyimpan default rate: %.2f", key, defaultRate)
		err = rdb.Set(ctx, "pricing:"+key, defaultRate, 24*time.Hour).Err()
		if err != nil {
			log.Printf("[Redis Warning] Gagal menyimpan default rate ke Redis: %v", err)
		}
		return defaultRate, nil
	} else if err != nil {
		return 0, err
	}

	rate, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, err
	}

	log.Printf("[Redis] Cache hit untuk key '%s': %.2f", key, rate)
	return rate, nil
}

