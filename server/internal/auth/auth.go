// Package auth provides pluggable bearer-token authentication for the
// management API and a Fiber middleware that enforces a required role.
package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

// ErrUnauthorized is returned when a token is missing or invalid.
var ErrUnauthorized = errors.New("unauthorized")

// AdminRole is the role required to access management endpoints.
const AdminRole = "admin"

// Principal is the authenticated caller.
type Principal struct {
	Subject string
	Role    string
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
		if p == nil || p.Role != role {
			return writeErr(c, fiber.StatusForbidden, "insufficient role", "forbidden")
		}
		c.Locals("principal", p)
		return c.Next()
	}
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
