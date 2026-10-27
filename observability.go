package operator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GetTelemetryInput selects the telemetry series to read. From and To are
// epoch milliseconds; Bucket is the bucket width in seconds (0 uses the
// server default); Agg is the bucket aggregate ("avg" when empty); FillLOCF
// carries the last observation forward across empty buckets.
type GetTelemetryInput struct {
	DeviceID string
	Metric   string
	From     time.Time
	To       time.Time
	Bucket   int
	Agg      string
	FillLOCF bool
}

// GetTelemetry returns bucketed telemetry points for a device metric.
func (c *Client) GetTelemetry(ctx context.Context, in GetTelemetryInput) ([]TelemetryPoint, error) {
	if in.DeviceID == "" {
		return nil, errors.New("operator: telemetry needs a device id")
	}
	v := url.Values{}
	if in.Metric != "" {
		v.Set("metric", in.Metric)
	}
	if !in.From.IsZero() {
		v.Set("from", in.From.UTC().Format(time.RFC3339Nano))
	}
	if !in.To.IsZero() {
		v.Set("to", in.To.UTC().Format(time.RFC3339Nano))
	}
	if in.Bucket > 0 {
		v.Set("bucket", fmt.Sprint(in.Bucket))
	}
	if in.Agg != "" {
		v.Set("agg", in.Agg)
	}
	if in.FillLOCF {
		v.Set("fill", "locf")
	}
	path := "{team}/devices/" + url.PathEscape(in.DeviceID) + "/telemetry"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var out struct {
		Items []TelemetryPoint `json:"items"`
	}
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Items, err
}

// ReadEvents reads up to limit events from the team's live event stream and
// returns them as a batch. The stream is server-sent events, which never
// terminates on its own, so the read is always bounded: it ends after limit
// events or when ctx is cancelled (the tool layer sets a timeout). A limit
// <= 0 reads a single event.
func (c *Client) ReadEvents(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 1
	}
	path, err := c.resolvePath(ctx, "{team}/events")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("operator: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/event-stream")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("operator: GET %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var env envelope
		raw, _ := io.ReadAll(res.Body)
		if len(bytes.TrimSpace(raw)) > 0 {
			_ = json.Unmarshal(raw, &env)
		}
		return nil, &ClientError{
			Method:     http.MethodGet,
			Path:       path,
			StatusCode: res.StatusCode,
			Message:    env.Message,
			RequestID:  requestID(env, res.Header),
		}
	}

	var events []Event
	var cur Event
	scanner := bufio.NewScanner(res.Body)
	// SSE data lines carry one JSON document each; allow generous frames.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			cur.Type = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			cur.Data = json.RawMessage(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		case line == "":
			if cur.Type != "" || cur.Data != nil {
				events = append(events, cur)
				cur = Event{}
				if len(events) >= limit {
					return events, nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return events, fmt.Errorf("operator: read event stream: %w", err)
	}
	return events, nil
}

// GetUsage returns the team's current usage against its plan limits.
func (c *Client) GetUsage(ctx context.Context) (Usage, error) {
	var out Usage
	err := c.do(ctx, http.MethodGet, "{team}/usage", nil, &out)
	return out, err
}

// GetAuditLog returns a page of audit-trail events for the token's team.
func (c *Client) GetAuditLog(ctx context.Context, page, limit int) (Page[AuditEvent], error) {
	var out Page[AuditEvent]
	err := c.do(ctx, http.MethodGet, "{team}/audit-events"+pageQuery(page, limit), nil, &out)
	return out, err
}
