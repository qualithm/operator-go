package operator

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestGetDeviceDecodesEveryField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		body  string
		check func(t *testing.T, d Device)
	}{
		{
			name: "a virtual, parked development device",
			body: `{"data":{"id":"dev_1","virtual":true,"parkedAt":"2026-10-28T00:00:00Z",` +
				`"capabilitiesDeclared":false,"level":"development",` +
				`"checks":[{"id":"credential-certificate","pass":false,"evidence":"a token credential"}]}}`,
			check: func(t *testing.T, d Device) {
				t.Helper()
				if d.Virtual == nil || !*d.Virtual {
					t.Errorf("Virtual = %v, want true", d.Virtual)
				}
				if d.ParkedAt == nil || !d.ParkedAt.Equal(time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)) {
					t.Errorf("ParkedAt = %v", d.ParkedAt)
				}
				if d.CapabilitiesDeclared == nil || *d.CapabilitiesDeclared {
					t.Errorf("CapabilitiesDeclared = %v, want false", d.CapabilitiesDeclared)
				}
				if d.Level != "development" {
					t.Errorf("Level = %q", d.Level)
				}
				want := DeviceCheck{ID: "credential-certificate", Pass: false, Evidence: "a token credential"}
				if len(d.Checks) != 1 || d.Checks[0] != want {
					t.Errorf("Checks = %+v", d.Checks)
				}
			},
		},
		{
			name: "a platform that sends none of them",
			body: `{"data":{"id":"dev_1","capabilitiesDeclared":null}}`,
			check: func(t *testing.T, d Device) {
				t.Helper()
				if d.Virtual != nil || d.ParkedAt != nil || d.CapabilitiesDeclared != nil {
					t.Errorf("optional fields = %v %v %v, want nil", d.Virtual, d.ParkedAt, d.CapabilitiesDeclared)
				}
				if d.Level != "" || d.Checks != nil {
					t.Errorf("Level, Checks = %q %v, want empty", d.Level, d.Checks)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, func(*http.Request) (*http.Response, error) {
				return jsonResponse(200, tt.body), nil
			})
			d, err := c.GetDevice(context.Background(), "dev_1")
			if err != nil {
				t.Fatalf("GetDevice: %v", err)
			}
			tt.check(t, d)
		})
	}
}
