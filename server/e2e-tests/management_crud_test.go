//go:build e2e

package e2e

import (
	"testing"

	"github.com/rxtech-lab/railway-wiki/e2e-tests/client"
)

// TestManagementCompanyCRUD exercises the full create→get→update→delete→404
// lifecycle on the management API for a representative resource (Company).
func TestManagementCompanyCRUD(t *testing.T) {
	admin := newAuthedClient(t)

	// Create.
	created, err := admin.AdminCreateCompanyWithResponse(ctx(t), client.CreateCompanyRequest{
		Name:      "E2E CRUD Co",
		ShortName: strPtr("CRUD"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.StatusCode() != 201 || created.JSON201 == nil || created.JSON201.Id == nil {
		t.Fatalf("create: expected 201 with id, got %d (%s)", created.StatusCode(), created.Body)
	}
	id := *created.JSON201.Id

	// Get.
	got, err := admin.AdminGetCompanyWithResponse(ctx(t), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.StatusCode() != 200 || got.JSON200 == nil || got.JSON200.Name != "E2E CRUD Co" {
		t.Fatalf("get: expected 200 with created data, got %d (%s)", got.StatusCode(), got.Body)
	}

	// Update (PUT replace).
	updated, err := admin.AdminUpdateCompanyWithResponse(ctx(t), id, client.UpdateCompanyRequest{
		Name:      "E2E CRUD Co (renamed)",
		ShortName: strPtr("CRUD2"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.StatusCode() != 200 || updated.JSON200 == nil || updated.JSON200.Name != "E2E CRUD Co (renamed)" {
		t.Fatalf("update: expected 200 with new name, got %d (%s)", updated.StatusCode(), updated.Body)
	}

	// Delete.
	del, err := admin.AdminDeleteCompanyWithResponse(ctx(t), id)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if del.StatusCode() != 204 {
		t.Fatalf("delete: expected 204, got %d (%s)", del.StatusCode(), del.Body)
	}

	// Get after delete -> 404.
	gone, err := admin.AdminGetCompanyWithResponse(ctx(t), id)
	if err != nil {
		t.Fatalf("get-after-delete: %v", err)
	}
	if gone.StatusCode() != 404 {
		t.Fatalf("get-after-delete: expected 404, got %d (%s)", gone.StatusCode(), gone.Body)
	}
}

// TestManagementStationCodeFilter creates a station and a station code that
// references it, then verifies the FK query filter (stationId) narrows the list.
func TestManagementStationCodeFilter(t *testing.T) {
	admin := newAuthedClient(t)

	// Create a parent station.
	st, err := admin.AdminCreateStationWithResponse(ctx(t), client.CreateStationRequest{
		Name:   "E2E Filter Station",
		NameEn: strPtr("E2E Filter Station"),
	})
	if err != nil {
		t.Fatalf("create station: %v", err)
	}
	if st.StatusCode() != 201 || st.JSON201 == nil || st.JSON201.Id == nil {
		t.Fatalf("create station: expected 201 with id, got %d (%s)", st.StatusCode(), st.Body)
	}
	stationID := *st.JSON201.Id
	t.Cleanup(func() { _, _ = admin.AdminDeleteStationWithResponse(ctx(t), stationID) })

	// Create a station code under it.
	sc, err := admin.AdminCreateStationCodeWithResponse(ctx(t), client.CreateStationCodeRequest{
		StationId: stationID,
		Code:      "E2E01",
	})
	if err != nil {
		t.Fatalf("create station code: %v", err)
	}
	if sc.StatusCode() != 201 || sc.JSON201 == nil || sc.JSON201.Id == nil {
		t.Fatalf("create station code: expected 201 with id, got %d (%s)", sc.StatusCode(), sc.Body)
	}
	codeID := *sc.JSON201.Id
	t.Cleanup(func() { _, _ = admin.AdminDeleteStationCodeWithResponse(ctx(t), codeID) })

	// List filtered by stationId -> must contain our code, and every returned
	// item must belong to that station.
	list, err := admin.AdminListStationCodesWithResponse(ctx(t), &client.AdminListStationCodesParams{
		StationId: &stationID,
	})
	if err != nil {
		t.Fatalf("list station codes: %v", err)
	}
	if list.StatusCode() != 200 || list.JSON200 == nil {
		t.Fatalf("list station codes: expected 200, got %d (%s)", list.StatusCode(), list.Body)
	}
	found := false
	for _, item := range list.JSON200.Items {
		if item.StationId != stationID {
			t.Fatalf("filter leaked a code for a different station: %v", item.StationId)
		}
		if item.Id != nil && *item.Id == codeID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created station code %s not found in stationId-filtered list", codeID)
	}
}

// TestManagementAuth verifies management endpoints reject missing/invalid auth
// and that malformed pagination cursors yield 400.
func TestManagementAuth(t *testing.T) {
	pub := newClient(t) // no Authorization header
	admin := newAuthedClient(t)

	t.Run("management list without token -> 401", func(t *testing.T) {
		resp, err := pub.AdminListCompaniesWithResponse(ctx(t), &client.AdminListCompaniesParams{})
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode() != 401 {
			t.Fatalf("expected 401, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})

	t.Run("management create without token -> 401", func(t *testing.T) {
		resp, err := pub.AdminCreateCompanyWithResponse(ctx(t), client.CreateCompanyRequest{Name: "nope"})
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode() != 401 {
			t.Fatalf("expected 401, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})

	t.Run("non-admin role list -> 403", func(t *testing.T) {
		viewer := newRoleClient(t, "e2e-viewer", "viewer")
		resp, err := viewer.AdminListCompaniesWithResponse(ctx(t), &client.AdminListCompaniesParams{})
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode() != 403 {
			t.Fatalf("expected 403 for non-admin role, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})

	t.Run("non-admin role create -> 403", func(t *testing.T) {
		viewer := newRoleClient(t, "e2e-viewer", "viewer")
		resp, err := viewer.AdminCreateCompanyWithResponse(ctx(t), client.CreateCompanyRequest{Name: "nope"})
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode() != 403 {
			t.Fatalf("expected 403 for non-admin role, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})

	t.Run("bad cursor -> 400", func(t *testing.T) {
		// Cursors are base64url; characters outside that alphabet fail to decode.
		resp, err := admin.AdminListCompaniesWithResponse(ctx(t), &client.AdminListCompaniesParams{
			Cursor: strPtr("!!!not-base64!!!"),
		})
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode() != 400 {
			t.Fatalf("expected 400 for bad cursor, got %d (%s)", resp.StatusCode(), resp.Body)
		}
	})
}
