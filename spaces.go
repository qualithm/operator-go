package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateSpaceInput is the request body for [Client.CreateSpace]. The platform
// assigns the space's initial name; rename it with [Client.UpdateSpace].
type CreateSpaceInput struct {
	Zone string `json:"zone"`
}

// ListSpaces returns a page of spaces for the token's team.
func (c *Client) ListSpaces(ctx context.Context, page, limit int) (Page[Space], error) {
	var out Page[Space]
	err := c.do(ctx, http.MethodGet, "/spaces"+pageQuery(page, limit), nil, &out)
	return out, err
}

// GetSpace retrieves a single space by ID.
func (c *Client) GetSpace(ctx context.Context, spaceID string) (Space, error) {
	var out Space
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/spaces/%s", spaceID), nil, &out)
	return out, err
}

// CreateSpace creates a space in the given device zone.
func (c *Client) CreateSpace(ctx context.Context, in CreateSpaceInput) (Space, error) {
	var out Space
	err := c.do(ctx, http.MethodPost, "/spaces", in, &out)
	return out, err
}

// UpdateSpace renames a space. This endpoint returns no resource body.
func (c *Client) UpdateSpace(ctx context.Context, spaceID, name string) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/spaces/%s", spaceID), map[string]string{"name": name}, nil)
}

// DeleteSpace deletes a space and cascades deletion to its devices.
func (c *Client) DeleteSpace(ctx context.Context, spaceID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/spaces/%s", spaceID), nil, nil)
}
