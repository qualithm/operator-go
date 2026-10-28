package operator

import (
	"context"
	"fmt"
	"net/http"
)

// ListAPITokens returns a page of API tokens for the token's team.
func (c *Client) ListAPITokens(ctx context.Context, cursor string, limit int) (Page[APIToken], error) {
	var out Page[APIToken]
	err := c.do(ctx, http.MethodGet, "{team}/api-tokens"+pageQuery(cursor, limit), nil, &out)
	return out, err
}

// RevokeAPIToken revokes an API token immediately.
func (c *Client) RevokeAPIToken(ctx context.Context, tokenID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("{team}/api-tokens/%s", tokenID), nil, nil)
}
