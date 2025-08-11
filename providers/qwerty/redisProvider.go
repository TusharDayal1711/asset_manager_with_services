package redisprovider

import (
	"asset/providers"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisDbProvider struct {
	client *redis.Client
}

func NewRedisProvider(addr string) providers.RedisProvider {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	return &RedisDbProvider{
		client: rdb,
	}
}

func (r *RedisDbProvider) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisDbProvider) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisDbProvider) Ping(ctx context.Context) error {
	pong, err := r.client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	fmt.Println("Redis Ping:", pong)
	return nil
}

func (r *RedisDbProvider) Close() error {
	return r.client.Close()
}
