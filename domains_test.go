package operator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTeams(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"team_1","name":"T","memberRole":"owner"}],"last":1}}`, &rec)
	page, err := c.ListTeams(context.Background(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/teams" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if len(page.Items) != 1 || page.Items[0].MemberRole != "owner" {
		t.Fatalf("page = %+v", page)
	}

	c = recClient(t, 201, `{"data":{"id":"team_2","name":"Team abcd1234"}}`, &rec)
	team, err := c.CreateTeam(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/teams" || rec.body != nil {
		t.Fatalf("request = %s %s body=%v", rec.method, rec.path, rec.body)
	}
	if team.ID != "team_2" {
		t.Fatalf("team = %+v", team)
	}

	c = recClient(t, 200, `{"data":{"id":"team_2"}}`, &rec)
	if _, err := c.GetTeam(context.Background(), "team_2"); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/teams/team_2" {
		t.Fatalf("path = %q", rec.path)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateTeam(context.Background(), "team_2", UpdateTeamInput{Name: "New"}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.body["name"] != "New" {
		t.Fatalf("request = %s body=%v", rec.method, rec.body)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteTeam(context.Background(), "team_2"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/teams/team_2" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}

	c = recClient(t, 200, `{"data":{"items":[{"deviceId":"dev_1","online":true,"lastSeenAt":123,"metrics":{"temp":21.5}}]}}`, &rec)
	snaps, err := c.GetTeamDeviceState(context.Background(), "team_2")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/teams/team_2/device-state" {
		t.Fatalf("path = %q", rec.path)
	}
	if len(snaps) != 1 || !snaps[0].Online || snaps[0].Metrics["temp"] != 21.5 {
		t.Fatalf("snaps = %+v", snaps)
	}
}

func TestMembers(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"mem_1","role":"manager"}],"last":1}}`, &rec)
	page, err := c.ListTeamMembers(context.Background(), "team_1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/teams/team_1/members" || len(page.Items) != 1 {
		t.Fatalf("path = %q page = %+v", rec.path, page)
	}

	c = recClient(t, 201, `{"data":{"id":"mem_2","status":"active"}}`, &rec)
	m, err := c.AddTeamMember(context.Background(), "team_1", AddTeamMemberInput{InviteID: "inv_9"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.body["inviteId"] != "inv_9" || m.Status != "active" {
		t.Fatalf("request = %s body=%v member=%+v", rec.method, rec.body, m)
	}

	c = recClient(t, 200, `{"data":{"id":"mem_1"}}`, &rec)
	if _, err := c.GetTeamMember(context.Background(), "team_1", "mem_1"); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/teams/team_1/members/mem_1" {
		t.Fatalf("path = %q", rec.path)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateTeamMember(context.Background(), "team_1", "mem_1", UpdateTeamMemberInput{Role: "guest"}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.body["role"] != "guest" {
		t.Fatalf("request = %s body=%v", rec.method, rec.body)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RemoveTeamMember(context.Background(), "team_1", "mem_1"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/teams/team_1/members/mem_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"mem_3","status":"invited"}],"last":1}}`, &rec)
	invites, err := c.ListTeamInvites(context.Background(), "team_1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/teams/team_1/invites" || len(invites.Items) != 1 {
		t.Fatalf("path = %q invites = %+v", rec.path, invites)
	}

	c = recClient(t, 201, `{"data":{"id":"mem_4","status":"invited"}}`, &rec)
	inv, err := c.CreateTeamInvite(context.Background(), "team_1", CreateTeamInviteInput{Email: "a@b.c", Role: "guest"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.body["email"] != "a@b.c" || rec.body["role"] != "guest" || inv.ID != "mem_4" {
		t.Fatalf("body=%v invite=%+v", rec.body, inv)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeTeamInvite(context.Background(), "team_1", "mem_4"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/teams/team_1/invites/mem_4" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestAutomations(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"auto_1","enabled":true}],"last":1}}`, &rec)
	page, err := c.ListAutomations(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations" || len(page.Items) != 1 || !page.Items[0].Enabled {
		t.Fatalf("path = %q page = %+v", rec.path, page)
	}

	payload := AutomationPayload{
		Trigger: AutomationTrigger{Type: "event", Config: json.RawMessage(`{"metric":"temp"}`)},
		Chain:   []AutomationChainStep{{Kind: "action", Type: "notification", Config: json.RawMessage(`{}`)}},
	}
	c = recClient(t, 201, `{"data":{"id":"auto_2","name":"A"}}`, &rec)
	a, err := c.CreateAutomation(context.Background(), CreateAutomationInput{Name: "A", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/automations" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if _, ok := rec.body["payload"].(map[string]any)["trigger"]; !ok {
		t.Fatalf("body missing trigger: %v", rec.body)
	}
	if a.ID != "auto_2" {
		t.Fatalf("automation = %+v", a)
	}

	c = recClient(t, 200, `{"data":{"id":"auto_2"}}`, &rec)
	if _, err := c.GetAutomation(context.Background(), "auto_2"); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations/auto_2" {
		t.Fatalf("path = %q", rec.path)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateAutomation(context.Background(), "auto_2", UpdateAutomationInput{Name: "B"}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.body["name"] != "B" {
		t.Fatalf("request = %s body=%v", rec.method, rec.body)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteAutomation(context.Background(), "auto_2"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/automations/auto_2" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}

	for _, tc := range []struct {
		name string
		call func(*Client) error
		path string
	}{
		{"enable", func(c *Client) error { _, err := c.EnableAutomation(context.Background(), "auto_2"); return err }, "/automations/auto_2/enable"},
		{"disable", func(c *Client) error { _, err := c.DisableAutomation(context.Background(), "auto_2"); return err }, "/automations/auto_2/disable"},
	} {
		c = recClient(t, 200, `{"data":{"id":"auto_2"}}`, &rec)
		if err := tc.call(c); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if rec.method != http.MethodPost || rec.path != tc.path {
			t.Fatalf("%s: request = %s %s", tc.name, rec.method, rec.path)
		}
	}

	c = recClient(t, 200, `{"data":{"id":"run_1"}}`, &rec)
	run, err := c.RunAutomation(context.Background(), "auto_2")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations/auto_2/run" || run.ID != "run_1" {
		t.Fatalf("path = %q run = %+v", rec.path, run)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"run_1","status":"succeeded"}],"last":1}}`, &rec)
	runs, err := c.ListAutomationRuns(context.Background(), "auto_2", 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations/auto_2/runs" || rec.query != "limit=5&page=2" || len(runs.Items) != 1 {
		t.Fatalf("path = %q query = %q runs = %+v", rec.path, rec.query, runs)
	}

	c = recClient(t, 201, `{"data":{"id":"auto_3"}}`, &rec)
	if _, err := c.CreateAutomationFromTemplate(context.Background(), CreateAutomationFromTemplateInput{Name: "T", Template: "tmpl_1", Params: map[string]any{"device": "dev_1"}}); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations/from-template" || rec.body["template"] != "tmpl_1" {
		t.Fatalf("path = %q body=%v", rec.path, rec.body)
	}

	c = recClient(t, 200, `{"data":[{"id":"tmpl_1","slots":[{"name":"device","kind":"device"}]}]}`, &rec)
	tmpls, err := c.ListAutomationTemplates(context.Background(), "dev_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/devices/dev_1/automation-templates" || len(tmpls) != 1 || tmpls[0].Slots[0].Kind != "device" {
		t.Fatalf("path = %q tmpls = %+v", rec.path, tmpls)
	}

	c = recClient(t, 200, `{"data":{"url":"https://x","secret":"s3cr3t"}}`, &rec)
	issued, err := c.CreateAutomationTriggerSecret(context.Background(), "auto_2")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/automations/auto_2/trigger-secret" || issued.Secret != "s3cr3t" {
		t.Fatalf("path = %q issued = %+v", rec.path, issued)
	}
}

// TestTriggerAutomationUsesSecret asserts the webhook trigger authenticates
// with the trigger secret, not the member token.
func TestTriggerAutomationUsesSecret(t *testing.T) {
	var auth string
	fn := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		auth = req.Header.Get("Authorization")
		return jsonResponse(200, `{"data":{"runId":"run_9"}}`), nil
	})
	c := newTestClient(t, fn)
	got, err := c.TriggerAutomation(context.Background(), "trig_secret", map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer trig_secret" {
		t.Fatalf("authorization = %q", auth)
	}
	if got.RunID != "run_9" {
		t.Fatalf("accepted = %+v", got)
	}
}

func TestDashboards(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"dash_1","spaceId":null}],"last":1}}`, &rec)
	page, err := c.ListDashboards(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/dashboards" || len(page.Items) != 1 || page.Items[0].SpaceID != nil {
		t.Fatalf("path = %q page = %+v", rec.path, page)
	}

	c = recClient(t, 201, `{"data":{"id":"dash_2"}}`, &rec)
	d, err := c.CreateDashboard(context.Background(), CreateDashboardInput{Name: "Board"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.body["name"] != "Board" || d.ID != "dash_2" {
		t.Fatalf("request = %s body=%v dash=%+v", rec.method, rec.body, d)
	}

	c = recClient(t, 200, `{"data":{"id":"dash_2"}}`, &rec)
	if _, err := c.GetDashboard(context.Background(), "dash_2"); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/dashboards/dash_2" {
		t.Fatalf("path = %q", rec.path)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateDashboard(context.Background(), "dash_2", UpdateDashboardInput{Name: "B2"}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.body["name"] != "B2" {
		t.Fatalf("request = %s body=%v", rec.method, rec.body)
	}

	c = recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteDashboard(context.Background(), "dash_2"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/dashboards/dash_2" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestDeviceCommands(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"cmd_1","status":"sent"}],"last":1}}`, &rec)
	page, err := c.ListDeviceCommands(context.Background(), "dev_1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/devices/dev_1/commands" || len(page.Items) != 1 {
		t.Fatalf("path = %q page = %+v", rec.path, page)
	}

	c = recClient(t, 202, `{"data":{"id":"cmd_2","duplicate":false}}`, &rec)
	q, err := c.SendDeviceCommand(context.Background(), "dev_1", SendDeviceCommandInput{Capability: "power", Value: true, DedupKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.body["capability"] != "power" || rec.body["dedupKey"] != "k1" || q.ID != "cmd_2" {
		t.Fatalf("request = %s body=%v queued=%+v", rec.method, rec.body, q)
	}

	c = recClient(t, 200, `{"data":[{"id":"cap_1","key":"power","type":"onoff"}]}`, &rec)
	caps, err := c.GetDeviceCapabilities(context.Background(), "dev_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/devices/dev_1/capabilities" || len(caps) != 1 || caps[0].Type != "onoff" {
		t.Fatalf("path = %q caps = %+v", rec.path, caps)
	}

	for _, suffix := range []string{"park", "unpark"} {
		c = recClient(t, 200, `{"message":"ok"}`, &rec)
		var err error
		if suffix == "park" {
			err = c.ParkDevice(context.Background(), "dev_1")
		} else {
			err = c.UnparkDevice(context.Background(), "dev_1")
		}
		if err != nil {
			t.Fatalf("%s: %v", suffix, err)
		}
		if rec.method != http.MethodPost || rec.path != "/devices/dev_1/"+suffix {
			t.Fatalf("%s: request = %s %s", suffix, rec.method, rec.path)
		}
	}
}

func TestObservability(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"items":[{"ts":1000,"value":2.5}]}}`, &rec)
	pts, err := c.GetTelemetry(context.Background(), GetTelemetryInput{DeviceID: "dev_1", Metric: "temp", From: 1, To: 2, Bucket: 60, Agg: "max", FillLOCF: true})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/telemetry" {
		t.Fatalf("path = %q", rec.path)
	}
	for _, want := range []string{"deviceId=dev_1", "metric=temp", "from=1", "to=2", "bucket=60", "agg=max", "fill=locf"} {
		if !strings.Contains(rec.query, want) {
			t.Fatalf("query %q missing %q", rec.query, want)
		}
	}
	if len(pts) != 1 || pts[0].TS != 1000 || pts[0].Value != 2.5 {
		t.Fatalf("points = %+v", pts)
	}

	c = recClient(t, 200, `{"data":{"deviceTotal":3,"deviceMax":10,"spaceTotal":2}}`, &rec)
	usage, err := c.GetUsage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/usage" || usage.DeviceTotal != 3 || usage.DeviceMax != 10 {
		t.Fatalf("path = %q usage = %+v", rec.path, usage)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"aud_1","action":"device.create"}],"last":1}}`, &rec)
	audit, err := c.GetAuditLog(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/audit" || len(audit.Items) != 1 || audit.Items[0].Action != "device.create" {
		t.Fatalf("path = %q audit = %+v", rec.path, audit)
	}
}

func TestReadEvents(t *testing.T) {
	stream := "event: device.connected\ndata: {\"deviceId\":\"dev_1\"}\n\nevent: telemetry\ndata: {\"temp\":21}\n\n"
	fn := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("accept = %q", got)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(stream)),
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		}, nil
	})
	c := newTestClient(t, fn)
	events, err := c.ReadEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Type != "device.connected" || events[1].Type != "telemetry" {
		t.Fatalf("events = %+v", events)
	}

	// The limit bounds the read even when the stream never ends.
	c = newTestClient(t, fn)
	events, err = c.ReadEvents(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("limited events = %+v", events)
	}

	// HTTP failures surface as ClientError, so the tool layer can classify them.
	c = recClient(t, 404, `{"message":"nope"}`, &recorder{})
	_, err = c.ReadEvents(context.Background(), 1)
	var ce *ClientError
	if !errors.As(err, &ce) || ce.StatusCode != 404 {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkspace(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"accountId":"acc_1","teamId":"team_1","memberRole":"owner"}}`, &rec)
	ws, err := c.GetWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/workspace" || ws.MemberRole != "owner" {
		t.Fatalf("path = %q ws = %+v", rec.path, ws)
	}

	c = recClient(t, 200, `{"data":{"id":"acc_1","email":"a@b.c"}}`, &rec)
	acct, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/account" || acct.Email != "a@b.c" {
		t.Fatalf("path = %q account = %+v", rec.path, acct)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"cap_1","key":"power"}],"last":1}}`, &rec)
	caps, err := c.ListCapabilities(context.Background(), ListCapabilitiesInput{Type: "onoff", Tag: "lamp", Key: "power", Page: 1, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/capabilities" || len(caps.Items) != 1 {
		t.Fatalf("path = %q caps = %+v", rec.path, caps)
	}
	for _, want := range []string{"type=onoff", "tag=lamp", "key=power"} {
		if !strings.Contains(rec.query, want) {
			t.Fatalf("query %q missing %q", rec.query, want)
		}
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"mem_1","role":"owner","teamName":"T"}],"last":1}}`, &rec)
	roles, err := c.ListRoles(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/roles" || len(roles.Items) != 1 || roles.Items[0].TeamName != "T" {
		t.Fatalf("path = %q roles = %+v", rec.path, roles)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"ses_1","thisDevice":true}],"last":1}}`, &rec)
	sessions, err := c.ListSessions(context.Background(), 0, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/sessions" || !strings.Contains(rec.query, "this_device=true") || len(sessions.Items) != 1 {
		t.Fatalf("path = %q query = %q sessions = %+v", rec.path, rec.query, sessions)
	}

	c = recClient(t, 200, `{"data":{"id":"ses_1","userAgent":"agent"}}`, &rec)
	sess, err := c.GetSession(context.Background(), "ses_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/sessions/ses_1" || sess.UserAgent != "agent" {
		t.Fatalf("path = %q session = %+v", rec.path, sess)
	}

	c = recClient(t, 200, `{"data":{"productUpdates":true,"securityAlerts":true}}`, &rec)
	prefs, err := c.GetCommunicationPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/account/communication-preferences" || !prefs["productUpdates"] {
		t.Fatalf("path = %q prefs = %+v", rec.path, prefs)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"spc_1","zone":"us"}],"last":1}}`, &rec)
	spaces, err := c.ListZoneSpaces(context.Background(), "us", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/zones/us/spaces" || len(spaces.Items) != 1 {
		t.Fatalf("path = %q spaces = %+v", rec.path, spaces)
	}
}

func TestBilling(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"tier":"pro","monthlyTotal":2900,"month":"2026-08","usage":{"devices":{"used":3,"limit":50}}}}`, &rec)
	sum, err := c.GetBillingSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/billing/summary" || sum.Tier != "pro" || sum.Usage.Devices.Used != 3 {
		t.Fatalf("path = %q summary = %+v", rec.path, sum)
	}

	c = recClient(t, 200, `{"data":{"current":1,"items":[{"id":"inv_1","status":"paid"}],"last":1}}`, &rec)
	invoices, err := c.ListInvoices(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/invoices" || len(invoices.Items) != 1 {
		t.Fatalf("path = %q invoices = %+v", rec.path, invoices)
	}

	c = recClient(t, 200, `{"data":{"from":"starter","to":"pro","amountDue":1200,"currency":"usd"}}`, &rec)
	preview, err := c.PreviewTierChange(context.Background(), "pro")
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/billing/tier/preview" || rec.body["tier"] != "pro" {
		t.Fatalf("request = %s %s body=%v", rec.method, rec.path, rec.body)
	}
	if preview.AmountDue != 1200 || preview.To != "pro" {
		t.Fatalf("preview = %+v", preview)
	}
}
