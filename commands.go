package operator

import (
	"context"
	"fmt"
	"net/http"
)

// SendDeviceCommandInput is the request body for [Client.SendDeviceCommand].
// DedupKey is caller-owned: a retried request with the same key does not
// command the device twice. Value is omitted for a trigger capability and
// required for every other commandable type.
type SendDeviceCommandInput struct {
	Capability string `json:"capability"`
	Value      any    `json:"value,omitempty"`
	DedupKey   string `json:"dedupKey"`
}

// ListDeviceCommands returns a page of command deliveries for a device.
func (c *Client) ListDeviceCommands(ctx context.Context, deviceID string, page, limit int) (Page[DeviceCommandSummary], error) {
	var out Page[DeviceCommandSummary]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/devices/%s/commands", deviceID)+pageQuery(page, limit), nil, &out)
	return out, err
}

// SendDeviceCommand queues a command for a device.
func (c *Client) SendDeviceCommand(ctx context.Context, deviceID string, in SendDeviceCommandInput) (DeviceCommandQueued, error) {
	var out DeviceCommandQueued
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/commands", deviceID), in, &out)
	return out, err
}

// GetDeviceCapabilities lists the capabilities a device declared in its
// connect-time manifest.
func (c *Client) GetDeviceCapabilities(ctx context.Context, deviceID string) ([]DeviceCapability, error) {
	var out []DeviceCapability
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/devices/%s/capabilities", deviceID), nil, &out)
	return out, err
}

// ParkDevice parks a device: it stops counting against the active-device
// limit while its credentials and history survive. Idempotent.
func (c *Client) ParkDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/park", deviceID), nil, nil)
}

// UnparkDevice returns a parked device to active. Idempotent.
func (c *Client) UnparkDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/unpark", deviceID), nil, nil)
}
