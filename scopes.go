package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateScopeInput is the request body for [Client.CreateScope]. A top-level
// scope takes a Zone; a child scope takes a ParentID and inherits its
// parent's zone.
type CreateScopeInput struct {
	Name     string `json:"name"`
	Zone     string `json:"zone,omitempty"`
	ParentID string `json:"parentId,omitempty"`
}

// ListScopes returns a page of scopes for the token's team, newest first.
func (c *Client) ListScopes(ctx context.Context, cursor string, limit int) (Page[Scope], error) {
	var out Page[Scope]
	err := c.do(ctx, http.MethodGet, "{team}/scopes"+pageQuery(cursor, limit), nil, &out)
	return out, err
}

// ListZoneScopes returns a page of the team's scopes in the given device zone.
func (c *Client) ListZoneScopes(ctx context.Context, zone, cursor string, limit int) (Page[Scope], error) {
	var out Page[Scope]
	err := c.do(ctx, http.MethodGet, "{team}/scopes"+filterQuery("zone", zone, cursor, limit), nil, &out)
	return out, err
}

// GetScope retrieves a single scope by ID.
func (c *Client) GetScope(ctx context.Context, scopeID string) (Scope, error) {
	var out Scope
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("{team}/scopes/%s", scopeID), nil, &out)
	return out, err
}

// CreateScope creates a scope: top-level in a zone, or under a parent. A zone
// unavailable in the current environment returns HTTP 403; callers should
// retry with an open zone rather than treat the pause as a fault. See the
// zones runbook in the Qualithm docs.
func (c *Client) CreateScope(ctx context.Context, in CreateScopeInput) (Scope, error) {
	var out Scope
	err := c.do(ctx, http.MethodPost, "{team}/scopes", in, &out)
	return out, err
}

// UpdateScope renames a scope. This endpoint returns no resource body.
func (c *Client) UpdateScope(ctx context.Context, scopeID, name string) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("{team}/scopes/%s", scopeID), map[string]string{"name": name}, nil)
}

// DeleteScope deletes an empty scope. The platform refuses it with HTTP 409
// while the scope still holds devices or scopes.
func (c *Client) DeleteScope(ctx context.Context, scopeID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("{team}/scopes/%s", scopeID), nil, nil)
}

// ListScopeSubtreeDevices returns a page of the devices in a scope and every
// scope under it. [Client.ListScopeDevices] lists the scope's own devices only.
func (c *Client) ListScopeSubtreeDevices(ctx context.Context, scopeID, cursor string, limit int) (Page[Device], error) {
	var out Page[Device]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("{team}/scopes/%s/devices", scopeID)+pageQuery(cursor, limit), nil, &out)
	return out, err
}
