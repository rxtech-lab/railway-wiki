//go:build e2e

package e2e

import "testing"

func TestHealth(t *testing.T) {
	c := newClient(t)

	resp, err := c.GetHealthWithResponse(ctx(t))
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode(), resp.Body)
	}
	if resp.JSON200 == nil {
		t.Fatalf("expected a health body, got none")
	}
}
