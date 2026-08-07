package qrinfra

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/qr"
	"github.com/redis/go-redis/v9"
)

// RedisQRStore stores QR tokens with a native TTL, so expired codes disappear
// without any cleanup job (RN-13).
type RedisQRStore struct {
	rdb *redis.Client
}

func NewRedisQRStore(rdb *redis.Client) qr.Store {
	return &RedisQRStore{rdb: rdb}
}

func tokenKey(code string) string  { return "neurons:qr:token:" + code }
func claimsKey(code string) string { return "neurons:qr:claims:" + code }

func (s *RedisQRStore) Save(ctx context.Context, token *qr.Token, ttl time.Duration) error {
	payload, err := json.Marshal(token)
	if err != nil {
		return qr.ErrInternal(err)
	}
	if err := s.rdb.Set(ctx, tokenKey(token.Code), payload, ttl).Err(); err != nil {
		return qr.ErrInternal(err)
	}
	return nil
}

func (s *RedisQRStore) Get(ctx context.Context, code string) (*qr.Token, error) {
	raw, err := s.rdb.Get(ctx, tokenKey(code)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, qr.ErrInvalid()
		}
		return nil, qr.ErrInternal(err)
	}

	var token qr.Token
	if err := json.Unmarshal(raw, &token); err != nil {
		return nil, qr.ErrInternal(err)
	}
	return &token, nil
}

func (s *RedisQRStore) Delete(ctx context.Context, code string) error {
	if err := s.rdb.Del(ctx, tokenKey(code), claimsKey(code)).Err(); err != nil {
		return qr.ErrInternal(err)
	}
	return nil
}

// ClaimOnce uses SADD, which returns 1 only for a member that was not present,
// giving a race-free "first claim wins" check.
func (s *RedisQRStore) ClaimOnce(
	ctx context.Context,
	code, claimerID string,
	ttl time.Duration,
) (bool, error) {
	key := claimsKey(code)
	added, err := s.rdb.SAdd(ctx, key, claimerID).Result()
	if err != nil {
		return false, qr.ErrInternal(err)
	}
	if err := s.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return false, qr.ErrInternal(err)
	}
	return added == 1, nil
}

func (s *RedisQRStore) ClaimCount(ctx context.Context, code string) (int, error) {
	count, err := s.rdb.SCard(ctx, claimsKey(code)).Result()
	if err != nil {
		return 0, qr.ErrInternal(err)
	}
	return int(count), nil
}
