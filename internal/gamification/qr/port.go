package qr

import (
	"context"
	"time"
)

// Store keeps QR tokens for their short lifetime. Redis is the natural backing
// store: entries expire on their own, so no cleanup job is needed (RN-13).
type Store interface {
	Save(ctx context.Context, token *Token, ttl time.Duration) error
	Get(ctx context.Context, code string) (*Token, error)
	// Delete burns a single-use code right after it is consumed.
	Delete(ctx context.Context, code string) error

	// ClaimOnce registers a claim of a grant code by one student and reports
	// whether this is the first time. It is atomic, so a double tap or a shared
	// screenshot cannot pay twice (§11.3).
	ClaimOnce(ctx context.Context, code, claimerID string, ttl time.Duration) (bool, error)
	// ClaimCount reports how many students already claimed a grant code.
	ClaimCount(ctx context.Context, code string) (int, error)
}
