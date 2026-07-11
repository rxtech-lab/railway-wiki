//go:build e2e

package e2e

import (
	"testing"

	"github.com/rxtech-lab/railway-wiki/e2e-tests/client"
)

// TestPublicRead seeds a company through the management API, then verifies the
// open public read endpoints surface it and that list responses use the cursor
// pagination envelope ({ items, pagination: { next } }).
func TestPublicRead(t *testing.T) {
	admin := newAuthedClient(t)
	pub := newClient(t)

	name := "E2E Public Read Railway"
	id := createCompany(t, admin, name)

	t.Run("get by id (public, no auth)", func(t *testing.T) {
		resp, err := pub.GetCompanyWithResponse(ctx(t), id)
		if err != nil {
			t.Fatalf("GET /api/companies/{id}: %v", err)
		}
		if resp.StatusCode() != 200 || resp.JSON200 == nil {
			t.Fatalf("expected 200 with body, got %d (%s)", resp.StatusCode(), resp.Body)
		}
		if resp.JSON200.Name != name {
			t.Fatalf("expected name %q, got %q", name, resp.JSON200.Name)
		}
	})

	t.Run("list + search + pagination shape", func(t *testing.T) {
		resp, err := pub.ListCompaniesWithResponse(ctx(t), &client.ListCompaniesParams{
			Q:     strPtr(name),
			Limit: intPtr(10),
		})
		if err != nil {
			t.Fatalf("GET /api/companies: %v", err)
		}
		if resp.StatusCode() != 200 || resp.JSON200 == nil {
			t.Fatalf("expected 200 with body, got %d (%s)", resp.StatusCode(), resp.Body)
		}
		// The paginated envelope must always be present.
		page := resp.JSON200
		found := false
		for _, comp := range page.Items {
			if comp.Id != nil && *comp.Id == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("created company %s not found in public list search for %q", id, name)
		}
		// pagination.next is a nullable cursor; nil is valid (last page).
		_ = page.Pagination.Next
	})

	t.Run("get unknown id returns 404", func(t *testing.T) {
		resp, err := pub.GetCompanyWithResponse(ctx(t), newUUID())
		if err != nil {
			t.Fatalf("GET /api/companies/{id}: %v", err)
		}
		if resp.StatusCode() != 404 {
			t.Fatalf("expected 404 for unknown id, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})
}
