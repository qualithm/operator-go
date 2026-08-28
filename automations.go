package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateAutomationInput is the request body for [Client.CreateAutomation].
// A nil SpaceID creates a team-wide automation.
type CreateAutomationInput struct {
	Name    string            `json:"name"`
	SpaceID *string           `json:"spaceId,omitempty"`
	Payload AutomationPayload `json:"payload"`
}

// UpdateAutomationInput is the request body for [Client.UpdateAutomation].
// Nil fields are left unchanged. Template/Params and Payload are mutually
// exclusive: setting Template re-expands the chain server-side from the
// template; the API does not accept clearing the space through this client.
type UpdateAutomationInput struct {
	Name     string             `json:"name,omitempty"`
	SpaceID  *string            `json:"spaceId,omitempty"`
	Payload  *AutomationPayload `json:"payload,omitempty"`
	Template string             `json:"template,omitempty"`
	Params   map[string]any     `json:"params,omitempty"`
}

// CreateAutomationFromTemplateInput is the request body for
// [Client.CreateAutomationFromTemplate]: a template ID plus slot values. The
// platform expands the chain server-side.
type CreateAutomationFromTemplateInput struct {
	Name     string         `json:"name"`
	Template string         `json:"template"`
	Params   map[string]any `json:"params,omitempty"`
}

// AutomationRunStarted is returned by [Client.RunAutomation]: the ID of the
// queued run.
type AutomationRunStarted struct {
	ID string `json:"id"`
}

// TriggerAccepted is returned by [Client.TriggerAutomation]: the ID of the
// run the webhook trigger queued.
type TriggerAccepted struct {
	RunID string `json:"runId"`
}

// IssuedTriggerSecret is returned by [Client.CreateAutomationTriggerSecret].
// URL and Secret are shown exactly once.
type IssuedTriggerSecret struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// ListAutomations returns a page of automations for the token's team.
func (c *Client) ListAutomations(ctx context.Context, page, limit int) (Page[Automation], error) {
	var out Page[Automation]
	err := c.do(ctx, http.MethodGet, "/automations"+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateAutomation creates an automation from a full payload.
func (c *Client) CreateAutomation(ctx context.Context, in CreateAutomationInput) (Automation, error) {
	var out Automation
	err := c.do(ctx, http.MethodPost, "/automations", in, &out)
	return out, err
}

// GetAutomation retrieves a single automation by ID.
func (c *Client) GetAutomation(ctx context.Context, automationID string) (Automation, error) {
	var out Automation
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/automations/%s", automationID), nil, &out)
	return out, err
}

// UpdateAutomation updates an automation's name, space, payload, or template
// params. This endpoint returns no resource body.
func (c *Client) UpdateAutomation(ctx context.Context, automationID string, in UpdateAutomationInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/automations/%s", automationID), in, nil)
}

// DeleteAutomation deletes an automation.
func (c *Client) DeleteAutomation(ctx context.Context, automationID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/automations/%s", automationID), nil, nil)
}

// EnableAutomation enables an automation and returns the updated resource.
func (c *Client) EnableAutomation(ctx context.Context, automationID string) (Automation, error) {
	var out Automation
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/automations/%s/enable", automationID), nil, &out)
	return out, err
}

// DisableAutomation disables an automation and returns the updated resource.
func (c *Client) DisableAutomation(ctx context.Context, automationID string) (Automation, error) {
	var out Automation
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/automations/%s/disable", automationID), nil, &out)
	return out, err
}

// RunAutomation queues a manual run of an automation.
func (c *Client) RunAutomation(ctx context.Context, automationID string) (AutomationRunStarted, error) {
	var out AutomationRunStarted
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/automations/%s/run", automationID), nil, &out)
	return out, err
}

// ListAutomationRuns returns a page of execution summaries for an automation.
func (c *Client) ListAutomationRuns(ctx context.Context, automationID string, page, limit int) (Page[AutomationRun], error) {
	var out Page[AutomationRun]
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/automations/%s/runs", automationID)+pageQuery(page, limit), nil, &out)
	return out, err
}

// CreateAutomationFromTemplate creates an automation by expanding a catalogue
// template server-side.
func (c *Client) CreateAutomationFromTemplate(ctx context.Context, in CreateAutomationFromTemplateInput) (Automation, error) {
	var out Automation
	err := c.do(ctx, http.MethodPost, "/automations/from-template", in, &out)
	return out, err
}

// ListAutomationTemplates returns the automation template catalogue filtered
// to what the given device declared it can do.
func (c *Client) ListAutomationTemplates(ctx context.Context, deviceID string) ([]AutomationTemplate, error) {
	var out []AutomationTemplate
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/devices/%s/automation-templates", deviceID), nil, &out)
	return out, err
}

// TriggerAutomation fires an automation's webhook trigger. Unlike every other
// method, it authenticates with the automation's trigger secret (issued by
// [Client.CreateAutomationTriggerSecret]), not the member API token. The
// optional context is passed to the chain.
func (c *Client) TriggerAutomation(ctx context.Context, secret string, triggerContext map[string]any) (TriggerAccepted, error) {
	var body any
	if len(triggerContext) > 0 {
		body = triggerContext
	}
	var out TriggerAccepted
	err := c.doAuth(ctx, http.MethodPost, "/automations/trigger", body, &out, secret)
	return out, err
}

// CreateAutomationTriggerSecret issues a webhook trigger secret for an
// automation. The URL and secret are shown exactly once.
func (c *Client) CreateAutomationTriggerSecret(ctx context.Context, automationID string) (IssuedTriggerSecret, error) {
	var out IssuedTriggerSecret
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/automations/%s/trigger-secret", automationID), nil, &out)
	return out, err
}
