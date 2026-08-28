package operator

import "encoding/json"

// Page is the standard paginated list envelope. Items may be nil in sparse
// pages, matching the API's `(T | undefined)[]` shape.
type Page[T any] struct {
	Current int  `json:"current"`
	Items   []*T `json:"items"`
	Last    int  `json:"last"`
}

// Authority is a device certificate authority (platform-generated or BYO).
type Authority struct {
	ID             string          `json:"id"`
	TeamID         string          `json:"teamId"`
	Name           string          `json:"name"`
	Kind           string          `json:"kind"` // "platform" | "byo"
	CertificatePEM string          `json:"certificatePem"`
	Fingerprint    string          `json:"fingerprint"`
	Status         string          `json:"status"` // "active" | "revoked"
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

// Enrollment is a one-time device enrollment code record.
type Enrollment struct {
	ID              string          `json:"id"`
	TeamID          string          `json:"teamId"`
	SpaceID         string          `json:"spaceId"`
	Label           string          `json:"label"`
	Status          string          `json:"status"` // "pending" | "claimed" | ...
	ClaimedDeviceID string          `json:"claimedDeviceId"`
	ClaimedAt       string          `json:"claimedAt"`
	ExpiresAt       string          `json:"expiresAt"`
	Payload         json.RawMessage `json:"payload"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// EnrollmentWithCode is returned by [Client.CreateEnrollment]. Code is the
// plaintext claim code, shown exactly once.
type EnrollmentWithCode struct {
	Enrollment Enrollment `json:"enrollment"`
	Code       string     `json:"code"`
}

// Credential is a device credential (token or cert).
type Credential struct {
	ID              string          `json:"id"`
	DeviceID        string          `json:"deviceId"`
	AuthorityID     string          `json:"authorityId"`
	Kind            string          `json:"kind"` // "token" | "cert"
	CertFingerprint string          `json:"certFingerprint"`
	CertSubject     string          `json:"certSubject"`
	Label           string          `json:"label"`
	Status          string          `json:"status"` // "active" | "revoked"
	ExpiresAt       string          `json:"expiresAt"`
	LastUsedAt      string          `json:"lastUsedAt"`
	RotatedFrom     string          `json:"rotatedFrom"`
	Payload         json.RawMessage `json:"payload"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// CredentialWithSecret is returned when minting or rotating a token
// credential. Secret is the plaintext bearer token, shown exactly once.
type CredentialWithSecret struct {
	Credential Credential `json:"credential"`
	Secret     string     `json:"secret"`
}

// CertCredential is returned when issuing an mTLS certificate credential.
// CertificatePEM (leaf) and CACertificatePEM are shown exactly once.
type CertCredential struct {
	Credential       Credential `json:"credential"`
	CertificatePEM   string     `json:"certificatePem"`
	CACertificatePEM string     `json:"caCertificatePem"`
}

// Device is a device resource with its space context.
type Device struct {
	ID        string          `json:"id"`
	SpaceID   string          `json:"spaceId"`
	Name      string          `json:"name"`
	TeamID    string          `json:"teamId"`
	SpaceName string          `json:"spaceName"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// Space is a space resource: the zone-scoped container devices live in.
type Space struct {
	ID          string          `json:"id"`
	TeamID      string          `json:"teamId"`
	Name        string          `json:"name"`
	Payload     json.RawMessage `json:"payload"`
	Zone        string          `json:"zone"`
	DeviceTotal int             `json:"deviceTotal"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

// APIToken is a member API token (metadata only; the secret is never listed).
type APIToken struct {
	ID         string `json:"id"`
	AccountID  string `json:"accountId"`
	TeamID     string `json:"teamId"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expiresAt"`
	LastUsedAt string `json:"lastUsedAt"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// APITokenWithSecret is returned by [Client.CreateAPIToken]. Secret is the
// plaintext token, shown exactly once.
type APITokenWithSecret struct {
	Token  APIToken `json:"token"`
	Secret string   `json:"secret"`
}

// Team is a team resource with the caller's membership context.
type Team struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Payload    json.RawMessage `json:"payload"`
	MemberID   string          `json:"memberId"`
	MemberRole string          `json:"memberRole"` // "owner" | "manager" | "guest"
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
}

// Member is a team membership with the member's account context.
type Member struct {
	ID           string `json:"id"`
	AccountID    string `json:"accountId"`
	TeamID       string `json:"teamId"`
	Role         string `json:"role"`   // "owner" | "manager" | "guest"
	Status       string `json:"status"` // "active" | "invited"
	AccountEmail string `json:"accountEmail"`
	AccountName  string `json:"accountName"`
	TeamName     string `json:"teamName"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// AutomationTrigger is the firing condition of an automation.
type AutomationTrigger struct {
	Type   string          `json:"type"` // "event" | "connectivity" | "schedule" | "webhook"
	Config json.RawMessage `json:"config"`
}

// AutomationChainStep is one step of an automation chain: a condition or an
// action.
type AutomationChainStep struct {
	Kind   string          `json:"kind"` // "condition" | "action"
	Type   string          `json:"type,omitempty"`
	Config json.RawMessage `json:"config"`
}

// AutomationTemplateReference links a template-built automation to the
// template and params it was expanded from.
type AutomationTemplateReference struct {
	ID      string          `json:"id"`
	Version int             `json:"version"`
	Params  json.RawMessage `json:"params"`
}

// AutomationPayload is the stored automation document: trigger plus chain.
type AutomationPayload struct {
	Trigger  AutomationTrigger            `json:"trigger"`
	Chain    []AutomationChainStep        `json:"chain"`
	Template *AutomationTemplateReference `json:"template,omitempty"`
}

// Automation is an automation resource with its space context.
type Automation struct {
	ID            string            `json:"id"`
	TeamID        string            `json:"teamId"`
	SpaceID       *string           `json:"spaceId"`
	Name          string            `json:"name"`
	Enabled       bool              `json:"enabled"`
	Payload       AutomationPayload `json:"payload"`
	SpaceName     *string           `json:"spaceName"`
	TemplateDrift bool              `json:"templateDrift,omitempty"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
}

// AutomationRunStep records the outcome of one chain step in a run.
type AutomationRunStep struct {
	Index      int    `json:"index"`
	Kind       string `json:"kind"`
	Type       string `json:"type,omitempty"`
	Outcome    string `json:"outcome"` // "passed" | "halted" | "succeeded" | "failed" | "skipped"
	Error      string `json:"error,omitempty"`
	DurationMS int    `json:"durationMs,omitempty"`
}

// AutomationRun is the customer-facing summary of one automation execution.
type AutomationRun struct {
	ID           string              `json:"id"`
	AutomationID string              `json:"automationId"`
	TriggerType  string              `json:"triggerType"`
	Status       string              `json:"status"` // "pending" | "succeeded" | "failed" | "skipped" | "limited"
	Steps        []AutomationRunStep `json:"steps"`
	LastError    *string             `json:"lastError"`
	TriggeredAt  string              `json:"triggeredAt"`
	ExecutedAt   *string             `json:"executedAt"`
}

// AutomationTemplate is one entry of the platform-owned automation catalogue.
type AutomationTemplate struct {
	ID          string                   `json:"id"`
	Version     int                      `json:"version"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Trigger     string                   `json:"trigger"`
	Slots       []AutomationTemplateSlot `json:"slots"`
}

// AutomationTemplateSlot names one typed parameter a template accepts.
type AutomationTemplateSlot struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // "channel" | "device" | "metric" | "control" | "recipient" | "range" | "schedule" | "value"
}

// Dashboard is a dashboard resource with its space context. Payload is the
// ordered widget array.
type Dashboard struct {
	ID        string          `json:"id"`
	TeamID    string          `json:"teamId"`
	SpaceID   *string         `json:"spaceId"`
	Name      string          `json:"name"`
	Payload   json.RawMessage `json:"payload"`
	SpaceName *string         `json:"spaceName"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// DeviceCommandSummary is the customer-facing view of one command delivery.
type DeviceCommandSummary struct {
	ID         string  `json:"id"`
	DeviceID   string  `json:"deviceId"`
	Source     string  `json:"source"` // "automation" | "dashboard" | "api"
	Topic      string  `json:"topic"`
	Status     string  `json:"status"` // "pending" | "sent" | "failed"
	LastError  *string `json:"lastError"`
	EnqueuedAt string  `json:"enqueuedAt"`
	SettledAt  *string `json:"settledAt"`
}

// DeviceCommandQueued is returned by [Client.SendDeviceCommand]. Duplicate
// means the dedup key was already seen and the device was not commanded again.
type DeviceCommandQueued struct {
	ID        string `json:"id"`
	Duplicate bool   `json:"duplicate"`
}

// DeviceCapability is one capability a device declared, including the
// operator-owned name and tags.
type DeviceCapability struct {
	ID       string          `json:"id"`
	DeviceID string          `json:"deviceId"`
	Key      string          `json:"key"`
	Type     string          `json:"type"` // "onoff" | "range" | "enum" | "trigger" | "sensor"
	Config   json.RawMessage `json:"config"`
	Name     *string         `json:"name"`
	Tags     []string        `json:"tags"`
}

// DeviceStateSnapshot is one device's latest sighting.
type DeviceStateSnapshot struct {
	DeviceID   string             `json:"deviceId"`
	SpaceID    string             `json:"spaceId"`
	Online     bool               `json:"online"`
	LastSeenAt int64              `json:"lastSeenAt"` // epoch milliseconds
	Metrics    map[string]float64 `json:"metrics"`
}

// TelemetryPoint is one bucketed telemetry reading. TS is epoch milliseconds
// at the bucket's start; Value is the aggregate over that bucket.
type TelemetryPoint struct {
	TS    int64   `json:"ts"`
	Value float64 `json:"value"`
}

// Usage is the team's current usage against its plan limits.
type Usage struct {
	DeviceTotal int64 `json:"deviceTotal"`
	DeviceMax   int   `json:"deviceMax"`
	SpaceTotal  int64 `json:"spaceTotal"`
}

// AuditEvent is one audit-trail row.
type AuditEvent struct {
	ID         string          `json:"id"`
	TeamID     string          `json:"teamId"`
	ActorKind  string          `json:"actorKind"`
	ActorID    *string         `json:"actorId"`
	Action     string          `json:"action"`
	TargetKind string          `json:"targetKind"`
	TargetID   string          `json:"targetId"`
	Metadata   json.RawMessage `json:"metadata"`
	OccurredAt string          `json:"occurredAt"`
}

// Workspace is the caller's current workspace context.
type Workspace struct {
	AccountID  string `json:"accountId"`
	TeamID     string `json:"teamId"`
	TeamName   string `json:"teamName"`
	MemberID   string `json:"memberId"`
	MemberRole string `json:"memberRole"`
	UpdatedAt  string `json:"updatedAt"`
}

// Account is the caller's account resource.
type Account struct {
	ID        string          `json:"id"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// RoleAssignment is one of the caller's team roles.
type RoleAssignment struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	TeamID    string `json:"teamId"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	TeamName  string `json:"teamName"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Session is one of the caller's authenticated sessions.
type Session struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	UserAgent   string `json:"userAgent"`
	IPAddress   string `json:"ipAddress"`
	MFAVerified bool   `json:"mfaVerified"`
	ThisDevice  bool   `json:"thisDevice"`
	CreatedAt   string `json:"createdAt"`
	LastUsedAt  string `json:"lastUsedAt"`
	ExpiresAt   string `json:"expiresAt"`
}

// BillingUsageDimension is one metered dimension's usage against its limit.
// Buffer is present only on the soft-ceiling events dimension.
type BillingUsageDimension struct {
	Used   int  `json:"used"`
	Limit  int  `json:"limit"`
	Buffer *int `json:"buffer"`
}

// BillingSummary is the team's billing state for the current month.
type BillingSummary struct {
	Tier         string `json:"tier"`
	MonthlyTotal int    `json:"monthlyTotal"`
	Month        string `json:"month"` // YYYY-MM, UTC
	LimitGrace   *struct {
		EndsAt string `json:"endsAt"`
	} `json:"limitGrace"`
	Limits struct {
		Devices           int `json:"devices"`
		RegisteredDevices int `json:"registeredDevices"`
		Events            int `json:"events"`
		Automations       int `json:"automations"`
	} `json:"limits"`
	Usage struct {
		Devices     BillingUsageDimension `json:"devices"`
		Registered  BillingUsageDimension `json:"registered"`
		Events      BillingUsageDimension `json:"events"`
		Automations BillingUsageDimension `json:"automations"`
	} `json:"usage"`
	Addons struct {
		Devices     int `json:"devices"`
		Events      int `json:"events"`
		Automations int `json:"automations"`
	} `json:"addons"`
}

// Invoice is one billing invoice.
type Invoice struct {
	ID        string          `json:"id"`
	TeamID    string          `json:"teamId"`
	Status    string          `json:"status"`
	TotalC    *string         `json:"totalC"`
	Currency  *string         `json:"currency"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

// TierChangePreview is what a tier change would cost, before it is confirmed.
type TierChangePreview struct {
	From               string `json:"from"`
	To                 string `json:"to"`
	AmountDue          int    `json:"amountDue"`
	Currency           string `json:"currency"`
	NextPaymentAttempt *int64 `json:"nextPaymentAttempt"`
}

// Event is one server-sent event from the team's event stream.
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
