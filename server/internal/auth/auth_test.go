package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type stubAuthenticator struct {
	principal *Principal
	err       error
}

func (a stubAuthenticator) Authenticate(context.Context, string) (*Principal, error) {
	return a.principal, a.err
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		principal  *Principal
		wantStatus int
	}{
		{name: "admin is allowed", principal: &Principal{Roles: []string{"member", AdminRole}}, wantStatus: http.StatusNoContent},
		{name: "non-admin is forbidden", principal: &Principal{Roles: []string{"member"}}, wantStatus: http.StatusForbidden},
		{name: "missing roles is forbidden", principal: &Principal{}, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(RequireRole(stubAuthenticator{principal: tt.principal}, AdminRole))
			app.Get("/admin", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set(fiber.HeaderAuthorization, "Bearer valid-token")
			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestStringSliceClaim(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{name: "JWT map claim", value: []any{"member", "admin"}, want: []string{"member", "admin"}},
		{name: "string slice", value: []string{"admin"}, want: []string{"admin"}},
		{name: "singular role is rejected", value: "admin", want: nil},
		{name: "non-string entries are ignored", value: []any{"admin", 42}, want: []string{"admin"}},
		{name: "missing claim", value: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, stringSliceClaim(tt.value))
		})
	}
}
