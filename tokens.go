package operator

import (
	"context"
	"fmt"
	"net/http"
)

// ListAPITokens returns a page of API tokens for the token's team.
func (c *Client) ListAPITokens(ctx context.Context, page, limit int) (Page[APIToken], error) {
	var out Page[APIToken]
	err := c.do(ctx, http.MethodGet, "/api-tokens"+pageQuery(page, limit), nil, &out)
	return out, err
}

// RevokeAPIToken revokes an API token immediately.
func (c *Client) RevokeAPIToken(ctx context.Context, tokenID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api-tokens/%s", tokenID), nil, nil)
}
