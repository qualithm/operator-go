package operator

import (
	"context"
	"fmt"
	"net/http"
)

// UpdateTeamInput is the request body for [Client.UpdateTeam].
type UpdateTeamInput struct {
	Name string `json:"name"`
}

// ListTeams returns a page of teams the token's member belongs to.
func (c *Client) ListTeams(ctx context.Context, page, limit int) (Page[Team], error) {
	var out Page[Team]
	err := c.do(ctx, http.MethodGet, "/teams"+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateTeam creates a team with the caller as owner. The platform assigns
// the initial name; rename it with [Client.UpdateTeam].
func (c *Client) CreateTeam(ctx context.Context) (Team, error) {
	var out Team
	err := c.do(ctx, http.MethodPost, "/teams", nil, &out)
	return out, err
}

// GetTeam retrieves a single team by ID.
func (c *Client) GetTeam(ctx context.Context, teamID string) (Team, error) {
	var out Team
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/teams/%s", teamID), nil, &out)
	return out, err
}

// UpdateTeam renames a team. This endpoint returns no resource body.
func (c *Client) UpdateTeam(ctx context.Context, teamID string, in UpdateTeamInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/teams/%s", teamID), in, nil)
}

// DeleteTeam deletes a team.
func (c *Client) DeleteTeam(ctx context.Context, teamID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/teams/%s", teamID), nil, nil)
}

// GetTeamDeviceState returns the latest state snapshot for every device in
// the team.
func (c *Client) GetTeamDeviceState(ctx context.Context, teamID string) ([]DeviceStateSnapshot, error) {
	var out struct {
		Items []DeviceStateSnapshot `json:"items"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/teams/%s/device-state", teamID), nil, &out)
	return out.Items, err
}
