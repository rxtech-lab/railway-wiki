// Package auth provides pluggable bearer-token authentication for the
// management API and a Fiber middleware that enforces a required role.
package auth

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

type principalContextKey struct{}

// ErrUnauthorized is returned when a token is missing or invalid.
var ErrUnauthorized = errors.New("unauthorized")

// AdminRole is the role required to access management endpoints.
const AdminRole = "admin"

// Principal is the authenticated caller.
type Principal struct {
	Subject string
	Roles   []string
}

// HasRole reports whether the authenticated caller carries role.
func (p *Principal) HasRole(role string) bool {
	return p != nil && slices.Contains(p.Roles, role)
}

// Authenticator validates a bearer token and returns the caller's principal.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

// RequireRole returns a Fiber middleware enforcing a valid bearer token whose
// principal has the required role. It writes the spec's {error, code} body on
// failure.
func RequireRole(a Authenticator, role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := bearer(c.Get(fiber.HeaderAuthorization))
		if token == "" {
			return writeErr(c, fiber.StatusUnauthorized, "missing bearer token", "unauthorized")
		}
		p, err := a.Authenticate(c.UserContext(), token)
		if err != nil {
			return writeErr(c, fiber.StatusUnauthorized, "invalid token", "unauthorized")
		}
		if !p.HasRole(role) {
			var receivedRoles []string
			if p != nil {
				receivedRoles = p.Roles
			}
			log.Printf(
				"management access denied: received_roles=%q expected_role=%q path=%q",
				receivedRoles,
				role,
				c.Path(),
			)
			return writeErr(c, fiber.StatusForbidden, "insufficient role", "forbidden")
		}
		c.Locals("principal", p)
		c.SetUserContext(context.WithValue(c.UserContext(), principalContextKey{}, p))
		return c.Next()
	}
}

// PrincipalFromContext returns the authenticated caller installed by
// RequireRole. Strict OpenAPI handlers receive this context.
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(*Principal)
	return p, ok && p != nil
}

func bearer(header string) string {
	const prefix = "Bearer "
	if len(header) >= len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

func writeErr(c *fiber.Ctx, status int, msg, code string) error {
	return c.Status(status).JSON(api.Error{Error: msg, Code: code})
}
