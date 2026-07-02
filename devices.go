package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateDeviceInput is the request body for [Client.CreateDevice].
type CreateDeviceInput struct {
	SpaceID string `json:"spaceId"`
}

// UpdateDeviceInput is the request body for [Client.UpdateDevice]. At least
// one field must be set.
type UpdateDeviceInput struct {
	Name    string `json:"name,omitempty"`
	SpaceID string `json:"spaceId,omitempty"`
}

// ListDevices returns a page of devices for the token's team.
func (c *Client) ListDevices(ctx context.Context, page, limit int) (Page[Device], error) {
	var out Page[Device]
	err := c.do(ctx, http.MethodGet, "/devices"+pageQuery(page, limit), nil, &out)
	return out, err
}

// ListSpaceDevices returns a page of devices in a specific space.
func (c *Client) ListSpaceDevices(ctx context.Context, spaceID string, page, limit int) (Page[Device], error) {
	var out Page[Device]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/spaces/%s/devices%s", spaceID, pageQuery(page, limit)), nil, &out)
	return out, err
}

// GetDevice retrieves a single device by ID.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (Device, error) {
	var out Device
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/devices/%s", deviceID), nil, &out)
	return out, err
}

// CreateDevice creates a device in a space.
func (c *Client) CreateDevice(ctx context.Context, in CreateDeviceInput) (Device, error) {
	var out Device
	err := c.do(ctx, http.MethodPost, "/devices", in, &out)
	return out, err
}

// UpdateDevice renames a device and/or moves it to another space in the same
// zone. This endpoint returns no resource body.
func (c *Client) UpdateDevice(ctx context.Context, deviceID string, in UpdateDeviceInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/devices/%s", deviceID), in, nil)
}

// DeleteDevice deletes a device and cascades revocation to its credentials.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/devices/%s", deviceID), nil, nil)
}
