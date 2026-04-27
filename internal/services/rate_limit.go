package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitService struct {
	client *redis.Client
	prefix string
}

func NewRateLimitService(client *redis.Client, env string) *RateLimitService {
	return &RateLimitService{
		client: client,
		prefix: fmt.Sprintf("%s:schooldesk:rl", env),
	}
}

func (s *RateLimitService) key(endpoint, subject string) string {
	return fmt.Sprintf("%s:%s:%s", s.prefix, endpoint, subject)
}

func (s *RateLimitService) Allow(ctx context.Context, endpoint, subject string, maxRequests int, window time.Duration) (bool, int, error) {
	key := s.key(endpoint, subject)
	count, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := s.client.Expire(ctx, key, window).Err(); err != nil {
			return false, 0, err
		}
	}
	remaining := maxRequests - int(count)
	return int(count) <= maxRequests, remaining, nil
}
