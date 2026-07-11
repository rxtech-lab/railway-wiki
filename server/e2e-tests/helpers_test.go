//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/rxtech-lab/railway-wiki/e2e-tests/client"
)

// newUUID returns a fresh random UUID (for negative "unknown id" lookups).
func newUUID() openapi_types.UUID { return uuid.New() }

// ctx returns a per-test context tied to the test's lifetime.
func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return c
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func int64Ptr(i int64) *int64 { return &i }

// createCompany creates a company via the management API and returns its id,
// registering a cleanup that deletes it. Fails the test on any error.
func createCompany(t *testing.T, c *client.ClientWithResponses, name string) openapi_types.UUID {
	t.Helper()
	resp, err := c.AdminCreateCompanyWithResponse(ctx(t), client.CreateCompanyRequest{
		Name:      name,
		ShortName: strPtr("E2E"),
		Country:   strPtr("JP"),
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	if resp.StatusCode() != 201 || resp.JSON201 == nil {
		t.Fatalf("create company: expected 201, got %d (%s)", resp.StatusCode(), resp.Body)
	}
	if resp.JSON201.Id == nil {
		t.Fatalf("create company: response missing id")
	}
	id := *resp.JSON201.Id
	t.Cleanup(func() {
		_, _ = c.AdminDeleteCompanyWithResponse(context.Background(), id)
	})
	return id
}
