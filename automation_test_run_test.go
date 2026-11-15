package operator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

const savedTestResult = `{"data":{
	"matched":true,
	"device":{"id":"dev_1","scopeId":"spc_1","online":true,"lastSeenAt":1763193600000},
	"reading":{"source":"device","metrics":{"temp":30},"seenAt":"2025-11-15T08:00:00.000Z"},
	"trigger":{"type":"event","matched":true},
	"compared":[{"index":0,"metric":"temp","op":"gt","expected":25,"actual":30,"passed":true}],
	"steps":[{"index":0,"kind":"condition","outcome":"passed"},{"index":1,"kind":"action","type":"notification","outcome":"skipped"}],
	"actions":[{"index":1,"type":"notification","channel":"email","config":{"to":"ops@example.com"}}],
	"decisions":[{"key":"channel","label":"Channel","value":"email","source":"automation","docsUrl":"https://docs.qualithm.com"}],
	"notification":{"sent":true,"channel":"email"}
}}`

func TestTestAutomation(t *testing.T) {
	t.Parallel()
	var rec recorder
	var key string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		key = req.Header.Get(IdempotencyHeader)
		return recClientTrip(&rec, req, 200, savedTestResult), nil
	})
	got, err := c.TestAutomation(context.Background(), "auto_1", AutomationTestOptions{DeviceID: "dev_1", Notify: true})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/teams/team_1/automations/auto_1/test" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if rec.body["deviceId"] != "dev_1" || rec.body["notify"] != true || rec.body["sample"] != nil {
		t.Fatalf("body = %v", rec.body)
	}
	if key != "" {
		t.Fatalf("a test sent an Idempotency-Key: %q", key)
	}
	if !got.Matched || got.Reason != "" || got.Trigger.Type != "event" || len(got.Steps) != 2 || len(got.Actions) != 1 {
		t.Fatalf("result = %+v", got)
	}
	if got.Device == nil || got.Device.LastSeenAt == nil || !got.Device.LastSeenAt.Equal(time.UnixMilli(1763193600000)) {
		t.Fatalf("device = %+v", got.Device)
	}
	if got.Reading == nil || got.Reading.SeenAt == nil || got.Reading.SeenAt.Format(time.RFC3339) != "2025-11-15T08:00:00Z" || got.Reading.Metrics["temp"] != 30 {
		t.Fatalf("reading = %+v", got.Reading)
	}
	if c := got.Compared[0]; c.Index == nil || *c.Index != 0 || c.Actual == nil || *c.Actual != 30 || !c.Passed {
		t.Fatalf("compared = %+v", got.Compared)
	}
	if got.Decisions[0].Key != "channel" || got.Notification == nil || !got.Notification.Sent {
		t.Fatalf("decisions = %+v notification = %+v", got.Decisions, got.Notification)
	}
}

func TestTestDraftAutomation(t *testing.T) {
	t.Parallel()
	var rec recorder
	c := recClient(t, 200, `{"data":{
		"matched":false,"reason":"no_reading",
		"device":{"id":"dev_1","scopeId":"spc_1","online":false,"lastSeenAt":null},
		"reading":null,
		"trigger":{"type":"connectivity","matched":false,"state":{"expected":"offline","actual":null}},
		"compared":[{"index":null,"metric":"temp","op":"lt","expected":5,"actual":null,"passed":false}],
		"steps":[],"actions":[],"decisions":[]
	}}`, &rec)
	got, err := c.TestDraftAutomation(context.Background(), TestDraftAutomationInput{
		Template:              "tmpl_1",
		Params:                map[string]any{"device": "dev_1"},
		AutomationTestOptions: AutomationTestOptions{Sample: &AutomationTestSample{Metrics: map[string]float64{"temp": 40}, State: "online"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/automations/test" {
		t.Fatalf("path = %q", rec.path)
	}
	sample, _ := rec.body["sample"].(map[string]any)
	if rec.body["template"] != "tmpl_1" || rec.body["name"] != nil || rec.body["payload"] != nil || sample["state"] != "online" {
		t.Fatalf("body = %v", rec.body)
	}
	if got.Matched || got.Reason != "no_reading" || got.Reading != nil || got.Device.LastSeenAt != nil || got.Notification != nil {
		t.Fatalf("result = %+v", got)
	}
	if got.Trigger.State == nil || got.Trigger.State.Actual != nil || got.Compared[0].Index != nil || got.Compared[0].Actual != nil {
		t.Fatalf("trigger = %+v compared = %+v", got.Trigger, got.Compared)
	}
}

func TestTestAutomationDryRun(t *testing.T) {
	t.Parallel()
	sent := 0
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		sent++
		return jsonResponse(200, `{"data":{"matched":true,"trigger":{"type":"manual","matched":true},"compared":[],"steps":[],"actions":[],"decisions":[]}}`), nil
	}, WithDryRun(true))

	if _, err := c.TestAutomation(context.Background(), "auto_1", AutomationTestOptions{DeviceID: "dev_1"}); err != nil {
		t.Fatalf("a read-only test should run in dry-run mode: %v", err)
	}
	_, err := c.TestDraftAutomation(context.Background(), TestDraftAutomationInput{Template: "tmpl_1", AutomationTestOptions: AutomationTestOptions{DeviceID: "dev_1", Notify: true}})
	var dry *DryRunError
	if !errors.As(err, &dry) || dry.Action.Path != "/v1/teams/team_1/automations/test" {
		t.Fatalf("a test that notifies should stop at dry run, got %v", err)
	}
	if sent != 1 {
		t.Fatalf("sent %d requests, want 1", sent)
	}
}

func TestAutomationTestBadTimes(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"device":        `{"data":{"device":{"id":"dev_1","lastSeenAt":"yesterday"}}}`,
		"reading":       `{"data":{"reading":{"source":"device","seenAt":true}}}`,
		"device shape":  `{"data":{"device":[]}}`,
		"reading shape": `{"data":{"reading":[]}}`,
	} {
		var rec recorder
		c := recClient(t, 200, body, &rec)
		if _, err := c.TestAutomation(context.Background(), "auto_1", AutomationTestOptions{DeviceID: "dev_1"}); err == nil {
			t.Errorf("%s: want a decode error", name)
		}
	}
}

// recClientTrip records req into rec as recClient does and answers with body.
func recClientTrip(rec *recorder, req *http.Request, status int, body string) *http.Response {
	rec.method = req.Method
	rec.path = req.URL.Path
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		rec.body = map[string]any{}
		_ = json.Unmarshal(raw, &rec.body)
	}
	return jsonResponse(status, body)
}
