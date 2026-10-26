package operator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// IdempotencyHeader is the header every POST carries, so the platform runs a
// create or action at most once per key and replays its first response to a
// retry.
const IdempotencyHeader = "Idempotency-Key"

type idempotencyKeyCtx struct{}

// WithIdempotencyKey returns a context whose POSTs send key as their
// Idempotency-Key. Without it, each POST sends a fresh key. Reuse one key when
// retrying the same call, so a retry after a lost response never acts twice.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyCtx{}, key)
}

// idempotencyKey is the context's key, or a fresh random one.
func idempotencyKey(ctx context.Context) (string, error) {
	if key, ok := ctx.Value(idempotencyKeyCtx{}).(string); ok && key != "" {
		return key, nil
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("operator: idempotency key: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
