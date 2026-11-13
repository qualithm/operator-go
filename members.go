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
	Role string `json:"role"` // "owner" | "manager" | "viewer"
}

// CreateTeamInviteInput is the request body for [Client.CreateTeamInvite].
type CreateTeamInviteInput struct {
	Emails []string `json:"emails"`
	Role   string   `json:"role"` // "owner" | "manager" | "viewer"
}

// TeamInviteResults is what [Client.CreateTeamInvite] did, one result per
// address in the order given.
type TeamInviteResults struct {
	Items []TeamInviteResult `json:"items"`
}

// inviteTokenInput is the request body for [Client.LookupInvite] and
// [Client.AcceptInvite].
type inviteTokenInput struct {
	Token string `json:"token"`
}

// ListTeamMembers returns a page of active members of the team.
func (c *Client) ListTeamMembers(ctx context.Context, teamID string, cursor string, limit int) (Page[Member], error) {
	var out Page[Member]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/teams/%s/members", teamID)+pageQuery(cursor, limit), nil, &out)
	return out, err
}

// AddTeamMember accepts an invite by its ID. Only the account whose email the
// invite names can accept it.
func (c *Client) AddTeamMember(ctx context.Context, teamID string, in AddTeamMemberInput) (Membership, error) {
	var out Membership
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/teams/%s/members", teamID), in, &out)
	return out, err
}

// GetTeamMember retrieves a single team member.
func (c *Client) GetTeamMember(ctx context.Context, teamID, memberID string) (Member, error) {
	var out Member
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/teams/%s/members/%s", teamID, memberID), nil, &out)
	return out, err
}

// UpdateTeamMember changes a member's role. This endpoint returns no resource
// body.
func (c *Client) UpdateTeamMember(ctx context.Context, teamID, memberID string, in UpdateTeamMemberInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/v1/teams/%s/members/%s", teamID, memberID), in, nil)
}

// RemoveTeamMember removes a member from the team.
func (c *Client) RemoveTeamMember(ctx context.Context, teamID, memberID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/teams/%s/members/%s", teamID, memberID), nil, nil)
}

// ListTeamInvites returns a page of pending invites to the team.
func (c *Client) ListTeamInvites(ctx context.Context, teamID string, cursor string, limit int) (Page[TeamInvite], error) {
	var out Page[TeamInvite]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/teams/%s/invites", teamID)+pageQuery(cursor, limit), nil, &out)
	return out, err
}

// CreateTeamInvite invites email addresses to the team with one role. Each
// address gets its own result: sent, already a member, or already invited.
func (c *Client) CreateTeamInvite(ctx context.Context, teamID string, in CreateTeamInviteInput) (TeamInviteResults, error) {
	var out TeamInviteResults
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/teams/%s/invites", teamID), in, &out)
	return out, err
}

// ResendTeamInvite mails a pending invite again with a new link; the old link
// stops working.
func (c *Client) ResendTeamInvite(ctx context.Context, teamID, inviteID string) (TeamInvite, error) {
	var out TeamInvite
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/teams/%s/invites/%s/resend", teamID, inviteID), nil, &out)
	return out, err
}

// RevokeTeamInvite withdraws a pending invite.
func (c *Client) RevokeTeamInvite(ctx context.Context, teamID, inviteID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/teams/%s/invites/%s", teamID, inviteID), nil, nil)
}

// LookupInvite returns the invite a link token names, and whether the caller
// may accept it.
func (c *Client) LookupInvite(ctx context.Context, token string) (InviteLink, error) {
	var out InviteLink
	err := c.do(ctx, http.MethodPost, "/v1/invites/lookup", inviteTokenInput{Token: token}, &out)
	return out, err
}

// AcceptInvite accepts the invite a link token names. Only the account whose
// email the invite names can accept it.
func (c *Client) AcceptInvite(ctx context.Context, token string) (Membership, error) {
	var out Membership
	err := c.do(ctx, http.MethodPost, "/v1/invites/accept", inviteTokenInput{Token: token}, &out)
	return out, err
}
