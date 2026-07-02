package operator

import (
	"context"
	"fmt"
	"net/http"
)

// MintCredentialInput is the request body for [Client.MintCredential] and
// [Client.RotateCredential].
type MintCredentialInput struct {
	Label string `json:"label,omitempty"`
	// ExpiresAt is an optional ISO 8601 timestamp in the future.
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// IssueCertInput is the request body for [Client.IssueCert].
type IssueCertInput struct {
	CSRPEM string `json:"csrPem"`
	Label  string `json:"label,omitempty"`
	// ExpiresInDays defaults to 30 server-side; range 1–3650.
	ExpiresInDays int `json:"expiresInDays,omitempty"`
}

// ListCredentials returns all credentials for a device.
func (c *Client) ListCredentials(ctx context.Context, deviceID string) ([]Credential, error) {
	var out []Credential
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/devices/%s/credentials", deviceID), nil, &out)
	return out, err
}

// MintCredential mints a token credential. The returned Secret is shown
// exactly once.
func (c *Client) MintCredential(ctx context.Context, deviceID string, in MintCredentialInput) (CredentialWithSecret, error) {
	var out CredentialWithSecret
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/credentials", deviceID), in, &out)
	return out, err
}

// IssueCert signs a device CSR with the team's platform CA. The returned
// CertificatePEM and CACertificatePEM are shown exactly once.
func (c *Client) IssueCert(ctx context.Context, deviceID string, in IssueCertInput) (CertCredential, error) {
	var out CertCredential
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/credentials/cert", deviceID), in, &out)
	return out, err
}

// RotateCredential rotates a token credential. When revoke is true the old
// credential is revoked immediately; otherwise it stays active so the device
// can finish swapping. The returned Secret is shown exactly once.
func (c *Client) RotateCredential(ctx context.Context, deviceID, credentialID string, in MintCredentialInput, revoke bool) (CredentialWithSecret, error) {
	path := fmt.Sprintf("/devices/%s/credentials/%s/rotate", deviceID, credentialID)
	if revoke {
		path += "?revoke=true"
	}
	var out CredentialWithSecret
	err := c.do(ctx, http.MethodPost, path, in, &out)
	return out, err
}

// RevokeCredential revokes a credential and drops any active session.
func (c *Client) RevokeCredential(ctx context.Context, deviceID, credentialID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/devices/%s/credentials/%s", deviceID, credentialID), nil, nil)
}
