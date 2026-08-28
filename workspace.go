package operator

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ListCapabilitiesInput filters [Client.ListCapabilities]. Zero values list
// every capability across the team's devices.
type ListCapabilitiesInput struct {
	Page  int
	Limit int
	Type  string // capability type: "onoff" | "range" | "enum" | "trigger" | "sensor"
	Tag   string
	Key   string
}

// GetWorkspace returns the caller's current workspace: team, membership, and
// role context.
func (c *Client) GetWorkspace(ctx context.Context) (Workspace, error) {
	var out Workspace
	err := c.do(ctx, http.MethodGet, "/workspace", nil, &out)
	return out, err
}

// GetAccount returns the caller's account.
func (c *Client) GetAccount(ctx context.Context) (Account, error) {
	var out Account
	err := c.do(ctx, http.MethodGet, "/account", nil, &out)
	return out, err
}

// ListCapabilities returns a page of device capabilities across the team.
func (c *Client) ListCapabilities(ctx context.Context, in ListCapabilitiesInput) (Page[DeviceCapability], error) {
	v := url.Values{}
	if in.Page > 0 {
		v.Set("page", fmt.Sprint(in.Page))
	}
	if in.Limit > 0 {
		v.Set("limit", fmt.Sprint(in.Limit))
	}
	if in.Type != "" {
		v.Set("type", in.Type)
	}
	if in.Tag != "" {
		v.Set("tag", in.Tag)
	}
	if in.Key != "" {
		v.Set("key", in.Key)
	}
	path := "/capabilities"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var out Page[DeviceCapability]
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// ListRoles returns a page of the caller's team role assignments.
func (c *Client) ListRoles(ctx context.Context, page, limit int) (Page[RoleAssignment], error) {
	var out Page[RoleAssignment]
	err := c.do(ctx, http.MethodGet, "/roles"+pageQuery(page, limit), nil, &out)
	return out, err
}

// ListSessions returns a page of the caller's authenticated sessions.
// ThisDeviceOnly narrows the list to the session the token belongs to.
func (c *Client) ListSessions(ctx context.Context, page, limit int, thisDeviceOnly bool) (Page[Session], error) {
	path := "/sessions" + pageQuery(page, limit)
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	if thisDeviceOnly {
		path += sep + "this_device=true"
	}
	var out Page[Session]
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// GetSession retrieves a single session by ID.
func (c *Client) GetSession(ctx context.Context, sessionID string) (Session, error) {
	var out Session
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/sessions/%s", sessionID), nil, &out)
	return out, err
}

// GetCommunicationPreferences returns the caller's email/push notification
// preferences as a preference-name → enabled map.
func (c *Client) GetCommunicationPreferences(ctx context.Context) (map[string]bool, error) {
	var out map[string]bool
	err := c.do(ctx, http.MethodGet, "/account/communication-preferences", nil, &out)
	return out, err
}

// ListZoneSpaces returns a page of spaces in the given device zone.
func (c *Client) ListZoneSpaces(ctx context.Context, zone string, page, limit int) (Page[Space], error) {
	var out Page[Space]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/zones/%s/spaces", zone)+pageQuery(page, limit), nil, &out)
	return out, err
}
