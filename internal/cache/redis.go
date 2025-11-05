package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	_ "github.com/joho/godotenv/autoload"
)

type Service interface {
	GetClient() *redis.Client
	Health() map[string]string
	Close() error
}

type service struct {
	client *redis.Client
}

var (
	redisAddr     = getEnv("REDIS_URL", "localhost:6379")
	redisPassword = getEnv("REDIS_PASSWORD", "")
	redisDB       = getEnvAsInt("REDIS_DB", 0)
	cacheInstance *service
)

func New() Service {
	if cacheInstance != nil {
		return cacheInstance
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Printf("[CACHE] Redis connection failed: %v", err)
		log.Println("[CACHE] Running without Redis cache")
		return nil
	}

	log.Println("[CACHE] Redis connected successfully")

	cacheInstance = &service{
		client: client,
	}

	return cacheInstance
}

func (s *service) GetClient() *redis.Client {
	return s.client
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	_, err := s.client.Ping(ctx).Result()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("redis down: %v", err)
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "Redis is healthy"

	poolStats := s.client.PoolStats()
	stats["hits"] = strconv.FormatUint(uint64(poolStats.Hits), 10)
	stats["misses"] = strconv.FormatUint(uint64(poolStats.Misses), 10)
	stats["timeouts"] = strconv.FormatUint(uint64(poolStats.Timeouts), 10)
	stats["total_conns"] = strconv.FormatUint(uint64(poolStats.TotalConns), 10)
	stats["idle_conns"] = strconv.FormatUint(uint64(poolStats.IdleConns), 10)
	stats["stale_conns"] = strconv.FormatUint(uint64(poolStats.StaleConns), 10)

	return stats
}

func (s *service) Close() error {
	log.Println("[CACHE] Disconnecting from Redis")
	return s.client.Close()
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
