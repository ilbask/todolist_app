package infrastructure

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps redis client with common operations
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client from environment variables
// Falls back to localhost:6379 if not specified
func NewRedisClient() *RedisClient {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")
	
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0, // default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis connection failed: %v. Caching will be disabled.", err)
		return &RedisClient{client: nil}
	}

	log.Println("✅ Redis connected successfully")
	return &RedisClient{client: client}
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r.client == nil {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key).Result()
}

// Set stores a value with TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if r.client == nil {
		return nil // Silently skip if Redis is unavailable
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del deletes a key
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	if r.client == nil {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// DelPattern deletes all keys matching a pattern
func (r *RedisClient) DelPattern(ctx context.Context, pattern string) error {
	if r.client == nil {
		return nil
	}

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// IsAvailable returns whether Redis is available
func (r *RedisClient) IsAvailable() bool {
	return r.client != nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}


