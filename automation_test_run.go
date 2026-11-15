package operator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AutomationTestSample is a reading to test against in place of a device's
// latest. An empty Topic stands for the trigger's own; an empty State for the
// device's ("online" or "offline").
type AutomationTestSample struct {
	Metrics map[string]float64 `json:"metrics,omitempty"`
	Topic   string             `json:"topic,omitempty"`
	State   string             `json:"state,omitempty"`
}

// AutomationTestOptions names what a test runs against: a device's latest
// reading, a sample, or both (the sample stands in for the reading). A test
// needs a device or a sample, unless the trigger pins a device. Notify also
// sends a test notification, to the caller alone, when one would fire.
type AutomationTestOptions struct {
	DeviceID string                `json:"deviceId,omitempty"`
	Sample   *AutomationTestSample `json:"sample,omitempty"`
	Notify   bool                  `json:"notify,omitempty"`
}

// TestDraftAutomationInput is the request body for [Client.TestDraftAutomation]:
// an unsaved automation as a create sends it, with the name optional, plus
// what to test it against. Payload and Template are mutually exclusive.
type TestDraftAutomationInput struct {
	Name     string             `json:"name,omitempty"`
	ScopeID  *string            `json:"scopeId,omitempty"`
	Payload  *AutomationPayload `json:"payload,omitempty"`
	Template string             `json:"template,omitempty"`
	Params   map[string]any     `json:"params,omitempty"`
	AutomationTestOptions
}

// AutomationTestDevice is the device a test evaluated.
type AutomationTestDevice struct {
	ID         string     `json:"id"`
	ScopeID    string     `json:"scopeId"`
	Online     bool       `json:"online"`
	LastSeenAt *time.Time `json:"-"`
}

// UnmarshalJSON accepts lastSeenAt as RFC 3339 or epoch milliseconds.
func (d *AutomationTestDevice) UnmarshalJSON(data []byte) error {
	type plain AutomationTestDevice
	var raw struct {
		plain
		LastSeenAt json.RawMessage `json:"lastSeenAt"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	at, err := parseTestTime(raw.LastSeenAt)
	if err != nil {
		return fmt.Errorf("operator: lastSeenAt: %w", err)
	}
	*d = AutomationTestDevice(raw.plain)
	d.LastSeenAt = at
	return nil
}

// AutomationTestReading is the reading a test evaluated: Source is "device"
// for a device's latest or "sample" for the caller's. SeenAt is nil for a
// sample.
type AutomationTestReading struct {
	Source  string             `json:"source"`
	Metrics map[string]float64 `json:"metrics"`
	SeenAt  *time.Time         `json:"-"`
	Topic   string             `json:"topic,omitempty"`
}

// UnmarshalJSON accepts seenAt as RFC 3339 or epoch milliseconds.
func (r *AutomationTestReading) UnmarshalJSON(data []byte) error {
	type plain AutomationTestReading
	var raw struct {
		plain
		SeenAt json.RawMessage `json:"seenAt"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	at, err := parseTestTime(raw.SeenAt)
	if err != nil {
		return fmt.Errorf("operator: seenAt: %w", err)
	}
	*r = AutomationTestReading(raw.plain)
	r.SeenAt = at
	return nil
}

// parseTestTime reads a time sent as an RFC 3339 string or epoch
// milliseconds; absent or null is nil.
func parseTestTime(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	if raw[0] == '"' {
		var t time.Time
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}
	var ms int64
	if err := json.Unmarshal(raw, &ms); err != nil {
		return nil, err
	}
	t := time.UnixMilli(ms).UTC()
	return &t, nil
}

// AutomationTestTriggerState is a connectivity trigger's watched state
// against the device's ("online", "offline", or empty when unknown).
type AutomationTestTriggerState struct {
	Expected json.RawMessage `json:"expected"`
	Actual   *string         `json:"actual"`
}

// AutomationTestTrigger is whether the trigger matched the reading.
type AutomationTestTrigger struct {
	Type    string                      `json:"type"` // "event" | "connectivity" | "schedule" | "webhook" | "manual"
	Matched bool                        `json:"matched"`
	State   *AutomationTestTriggerState `json:"state,omitempty"`
}

// AutomationTestComparison is one comparison a test made against the
// reading. A nil Index is the trigger; a nil Actual means the reading lacks
// the metric.
type AutomationTestComparison struct {
	Index    *int     `json:"index"`
	Metric   string   `json:"metric"`
	Op       string   `json:"op"` // "eq" | "ne" | "gt" | "gte" | "lt" | "lte"
	Expected float64  `json:"expected"`
	Actual   *float64 `json:"actual"`
	Passed   bool     `json:"passed"`
}

// AutomationTestAction is an action the automation would take; a test takes
// none. A signing secret is never shown.
type AutomationTestAction struct {
	Index   int             `json:"index"`
	Type    string          `json:"type"` // "device-command" | "notification" | "outbound-webhook"
	Channel json.RawMessage `json:"channel,omitempty"`
	Config  json.RawMessage `json:"config"`
}

// AutomationTestNotification is the test notification, when one was asked
// for: whether it was sent and on which channel.
type AutomationTestNotification struct {
	Sent    bool    `json:"sent"`
	Channel *string `json:"channel"`
}

// AutomationTest is what an automation would do with a reading. Nothing is
// run, recorded or counted. Reason is empty when Matched is true, else one of
// "no_reading", "device_offline", "out_of_scope", "trigger_not_matched",
// "condition_halted", "action_failed" or "disabled".
type AutomationTest struct {
	Matched      bool                        `json:"matched"`
	Reason       string                      `json:"reason,omitempty"`
	Device       *AutomationTestDevice       `json:"device"`
	Reading      *AutomationTestReading      `json:"reading"`
	Trigger      AutomationTestTrigger       `json:"trigger"`
	Compared     []AutomationTestComparison  `json:"compared"`
	Steps        []AutomationRunStep         `json:"steps"`
	Actions      []AutomationTestAction      `json:"actions"`
	Decisions    []Decision                  `json:"decisions"`
	Notification *AutomationTestNotification `json:"notification,omitempty"`
}

// TestAutomation tests a saved automation against a device's latest reading
// or a sample, without running it. A test is read-only unless opts.Notify is
// set, so dry-run mode lets it through, and it sends no Idempotency-Key.
func (c *Client) TestAutomation(ctx context.Context, automationID string, opts AutomationTestOptions) (AutomationTest, error) {
	var out AutomationTest
	err := c.doTest(ctx, fmt.Sprintf("{team}/automations/%s/test", automationID), opts, opts.Notify, &out)
	return out, err
}

// TestDraftAutomation tests an unsaved automation against a device's latest
// reading or a sample. Like [Client.TestAutomation], it is read-only unless
// Notify is set and sends no Idempotency-Key.
func (c *Client) TestDraftAutomation(ctx context.Context, in TestDraftAutomationInput) (AutomationTest, error) {
	var out AutomationTest
	err := c.doTest(ctx, "{team}/automations/test", in, in.Notify, &out)
	return out, err
}

func (c *Client) doTest(ctx context.Context, path string, body any, notify bool, out any) error {
	return c.send(ctx, http.MethodPost, path, body, out, requestOptions{readOnly: !notify, noIdempotencyKey: true})
}
