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
