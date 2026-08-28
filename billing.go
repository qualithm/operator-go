package operator

import (
	"context"
	"net/http"
)

// GetBillingSummary returns the team's billing state for the current month:
// tier, limits, usage per dimension, and add-ons.
func (c *Client) GetBillingSummary(ctx context.Context) (BillingSummary, error) {
	var out BillingSummary
	err := c.do(ctx, http.MethodGet, "/billing/summary", nil, &out)
	return out, err
}

// ListInvoices returns a page of the team's invoices.
func (c *Client) ListInvoices(ctx context.Context, page, limit int) (Page[Invoice], error) {
	var out Page[Invoice]
	err := c.do(ctx, http.MethodGet, "/invoices"+pageQuery(page, limit), nil, &out)
	return out, err
}

// PreviewTierChange prices a move to the given paid tier without applying it.
// Read-only despite the POST method.
func (c *Client) PreviewTierChange(ctx context.Context, tier string) (TierChangePreview, error) {
	var out TierChangePreview
	err := c.do(ctx, http.MethodPost, "/billing/tier/preview", map[string]string{"tier": tier}, &out)
	return out, err
}
