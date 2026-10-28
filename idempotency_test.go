package operator

import (
	"context"
	"net/http"
	"testing"
)

func TestIdempotencyKey(t *testing.T) {
	t.Parallel()
	var keys []string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		keys = append(keys, req.Header.Get(IdempotencyHeader))
		return jsonResponse(201, `{"data":{"id":"spc_1"}}`), nil
	})

	for range 2 {
		if _, err := c.CreateSpace(context.Background(), CreateSpaceInput{Zone: "sg-sin-a"}); err != nil {
			t.Fatal(err)
		}
	}
	if len(keys[0]) != 32 || keys[0] == keys[1] {
		t.Fatalf("each POST should send its own fresh key, got %q", keys)
	}

	ctx := WithIdempotencyKey(context.Background(), "retry-1")
	for range 2 {
		if _, err := c.CreateSpace(ctx, CreateSpaceInput{Zone: "sg-sin-a"}); err != nil {
			t.Fatal(err)
		}
	}
	if keys[2] != "retry-1" || keys[3] != "retry-1" {
		t.Fatalf("a retry should reuse the caller's key, got %q", keys[2:])
	}

	if _, err := c.ListSpaces(context.Background(), "", 0); err != nil {
		t.Fatal(err)
	}
	if keys[4] != "" {
		t.Fatalf("a GET sent a key: %q", keys[4])
	}
}
