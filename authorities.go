package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateAuthorityInput is the request body for [Client.CreateAuthority].
type CreateAuthorityInput struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // "platform" | "byo"
	// CertificatePEM is required when Kind is "byo".
	CertificatePEM string `json:"certificatePem,omitempty"`
}

// ListAuthorities returns a page of authorities for the token's team.
func (c *Client) ListAuthorities(ctx context.Context, page, limit int) (Page[Authority], error) {
	var out Page[Authority]
	err := c.do(ctx, http.MethodGet, "/authorities"+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateAuthority registers a platform-generated or BYO authority.
func (c *Client) CreateAuthority(ctx context.Context, in CreateAuthorityInput) (Authority, error) {
	var out Authority
	err := c.do(ctx, http.MethodPost, "/authorities", in, &out)
	return out, err
}

// RevokeAuthority revokes an authority and cascades revocation to the
// credentials it issued.
func (c *Client) RevokeAuthority(ctx context.Context, authorityID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/authorities/%s", authorityID), nil, nil)
}
