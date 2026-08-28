package operator

import (
	"context"
	"fmt"
	"net/http"
)

// AddTeamMemberInput is the request body for [Client.AddTeamMember]: the
// caller accepts their own invitation to the team.
type AddTeamMemberInput struct {
	InviteID string `json:"inviteId"`
}

// UpdateTeamMemberInput is the request body for [Client.UpdateTeamMember].
type UpdateTeamMemberInput struct {
	Role string `json:"role"` // "owner" | "manager" | "guest"
}

// CreateTeamInviteInput is the request body for [Client.CreateTeamInvite].
type CreateTeamInviteInput struct {
	Email string `json:"email"`
	Role  string `json:"role"` // "owner" | "manager" | "guest"
}

// ListTeamMembers returns a page of active members of the team.
func (c *Client) ListTeamMembers(ctx context.Context, teamID string, page, limit int) (Page[Member], error) {
	var out Page[Member]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/teams/%s/members", teamID)+pageQuery(page, limit), nil, &out)
	return out, err
}

// AddTeamMember activates the caller's invited membership from an invite ID.
func (c *Client) AddTeamMember(ctx context.Context, teamID string, in AddTeamMemberInput) (Member, error) {
	var out Member
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/teams/%s/members", teamID), in, &out)
	return out, err
}

// GetTeamMember retrieves a single team member.
func (c *Client) GetTeamMember(ctx context.Context, teamID, memberID string) (Member, error) {
	var out Member
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/teams/%s/members/%s", teamID, memberID), nil, &out)
	return out, err
}

// UpdateTeamMember changes a member's role. This endpoint returns no resource
// body.
func (c *Client) UpdateTeamMember(ctx context.Context, teamID, memberID string, in UpdateTeamMemberInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/teams/%s/members/%s", teamID, memberID), in, nil)
}

// RemoveTeamMember removes a member from the team.
func (c *Client) RemoveTeamMember(ctx context.Context, teamID, memberID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/teams/%s/members/%s", teamID, memberID), nil, nil)
}

// ListTeamInvites returns a page of pending invitations to the team.
func (c *Client) ListTeamInvites(ctx context.Context, teamID string, page, limit int) (Page[Member], error) {
	var out Page[Member]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/teams/%s/invites", teamID)+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateTeamInvite invites an account to the team by email.
func (c *Client) CreateTeamInvite(ctx context.Context, teamID string, in CreateTeamInviteInput) (Member, error) {
	var out Member
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/teams/%s/invites", teamID), in, &out)
	return out, err
}

// RevokeTeamInvite revokes a pending invitation.
func (c *Client) RevokeTeamInvite(ctx context.Context, teamID, inviteID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/teams/%s/invites/%s", teamID, inviteID), nil, nil)
}
