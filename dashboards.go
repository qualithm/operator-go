package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateDashboardInput is the request body for [Client.CreateDashboard]. A nil
// SpaceID creates a team-wide dashboard; Payload is the ordered widget array
// (nil starts empty).
type CreateDashboardInput struct {
	Name    string          `json:"name"`
	SpaceID *string         `json:"spaceId,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// UpdateDashboardInput is the request body for [Client.UpdateDashboard]. Nil
// fields are left unchanged; the API does not accept clearing the space
// through this client.
type UpdateDashboardInput struct {
	Name    string          `json:"name,omitempty"`
	SpaceID *string         `json:"spaceId,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ListDashboards returns a page of dashboards for the token's team.
func (c *Client) ListDashboards(ctx context.Context, page, limit int) (Page[Dashboard], error) {
	var out Page[Dashboard]
	err := c.do(ctx, http.MethodGet, "/dashboards"+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateDashboard creates a dashboard.
func (c *Client) CreateDashboard(ctx context.Context, in CreateDashboardInput) (Dashboard, error) {
	var out Dashboard
	err := c.do(ctx, http.MethodPost, "/dashboards", in, &out)
	return out, err
}

// GetDashboard retrieves a single dashboard by ID.
func (c *Client) GetDashboard(ctx context.Context, dashboardID string) (Dashboard, error) {
	var out Dashboard
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/dashboards/%s", dashboardID), nil, &out)
	return out, err
}

// UpdateDashboard updates a dashboard's name, space, or widgets. This
// endpoint returns no resource body.
func (c *Client) UpdateDashboard(ctx context.Context, dashboardID string, in UpdateDashboardInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/dashboards/%s", dashboardID), in, nil)
}

// DeleteDashboard deletes a dashboard.
func (c *Client) DeleteDashboard(ctx context.Context, dashboardID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/dashboards/%s", dashboardID), nil, nil)
}
