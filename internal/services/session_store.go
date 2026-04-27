package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client *redis.Client
	prefix string
}

func NewSessionStore(client *redis.Client, env string) *SessionStore {
	return &SessionStore{
		client: client,
		prefix: fmt.Sprintf("%s:schooldesk:sess", env),
	}
}

func (s *SessionStore) jtiKey(jti string) string {
	return fmt.Sprintf("%s:jti:%s", s.prefix, jti)
}

func (s *SessionStore) refreshKey(token string) string {
	return fmt.Sprintf("%s:refresh:%s", s.prefix, token)
}

func (s *SessionStore) RevokeJTI(ctx context.Context, jti string, ttl time.Duration) error {
	return s.client.Set(ctx, s.jtiKey(jti), "1", ttl).Err()
}

func (s *SessionStore) IsJTIRevoked(ctx context.Context, jti string) (bool, error) {
	exists, err := s.client.Exists(ctx, s.jtiKey(jti)).Result()
	return exists > 0, err
}

func (s *SessionStore) StoreRefreshToken(ctx context.Context, token string, payload string, ttl time.Duration) error {
	return s.client.Set(ctx, s.refreshKey(token), payload, ttl).Err()
}

func (s *SessionStore) GetRefreshToken(ctx context.Context, token string) (string, error) {
	return s.client.Get(ctx, s.refreshKey(token)).Result()
}

func (s *SessionStore) RevokeRefreshToken(ctx context.Context, token string) error {
	return s.client.Del(ctx, s.refreshKey(token)).Err()
}
