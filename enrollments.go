package operator

import (
	"context"
	"fmt"
	"net/http"
)

// CreateEnrollmentInput is the request body for [Client.CreateEnrollment].
type CreateEnrollmentInput struct {
	SpaceID string `json:"spaceId"`
	Label   string `json:"label,omitempty"`
	// ExpiresInMinutes defaults to 1440 (24h) server-side; range 1–525600.
	ExpiresInMinutes int `json:"expiresInMinutes,omitempty"`
}

// ListEnrollments returns a page of enrollments for the token's team.
func (c *Client) ListEnrollments(ctx context.Context, cursor string, limit int) (Page[Enrollment], error) {
	var out Page[Enrollment]
	err := c.do(ctx, http.MethodGet, "{team}/enrollments"+pageQuery(cursor, limit), nil, &out)
	return out, err
}

// CreateEnrollment creates a one-time enrollment code. The returned Code is
// shown exactly once.
func (c *Client) CreateEnrollment(ctx context.Context, in CreateEnrollmentInput) (EnrollmentWithCode, error) {
	var out EnrollmentWithCode
	err := c.do(ctx, http.MethodPost, "{team}/enrollments", in, &out)
	return out, err
}

// RevokeEnrollment revokes a pending enrollment.
func (c *Client) RevokeEnrollment(ctx context.Context, enrollmentID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("{team}/enrollments/%s", enrollmentID), nil, nil)
}
