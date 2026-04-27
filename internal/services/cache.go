package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client *redis.Client
	prefix string
}

func NewCacheService(client *redis.Client, env string) *CacheService {
	return &CacheService{
		client: client,
		prefix: fmt.Sprintf("%s:schooldesk:cache", env),
	}
}

func (s *CacheService) key(key string) string {
	return fmt.Sprintf("%s:%s", s.prefix, key)
}

func (s *CacheService) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, s.key(key)).Result()
}

func (s *CacheService) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, s.key(key), value, ttl).Err()
}

func (s *CacheService) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	redisKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		redisKeys = append(redisKeys, s.key(key))
	}
	return s.client.Del(ctx, redisKeys...).Err()
}

func (s *CacheService) DeleteByPrefix(ctx context.Context, prefix string) error {
	pattern := s.key(prefix) + "*"
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
