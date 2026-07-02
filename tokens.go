package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateAPITokenInput is the request body for [Client.CreateAPIToken].
type CreateAPITokenInput struct {
	Name string `json:"name,omitempty"`
	// ExpiresInDays defaults to 90 server-side; range 1–365.
	ExpiresInDays int `json:"expiresInDays,omitempty"`
}

// ListAPITokens returns a page of API tokens for the token's team.
func (c *Client) ListAPITokens(ctx context.Context, page, limit int) (Page[APIToken], error) {
	var out Page[APIToken]
	err := c.do(ctx, http.MethodGet, "/api-tokens"+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateAPIToken mints a new API token. The returned Secret is shown exactly
// once.
func (c *Client) CreateAPIToken(ctx context.Context, in CreateAPITokenInput) (APITokenWithSecret, error) {
	var out APITokenWithSecret
	err := c.do(ctx, http.MethodPost, "/api-tokens", in, &out)
	return out, err
}

// RevokeAPIToken revokes an API token immediately.
func (c *Client) RevokeAPIToken(ctx context.Context, tokenID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api-tokens/%s", tokenID), nil, nil)
}
