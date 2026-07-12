package server

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/overpass"
	"github.com/rxtech-lab/railway-wiki/internal/repo"
)

// ErrorMiddleware centralizes error-to-HTTP-status mapping for the strict
// handlers.
//
// oapi-codegen's generated wrapper turns ANY error returned by a handler into a
// bare HTTP 400, discarding status. This StrictMiddlewareFunc intercepts the
// handler result: on error it writes the spec's {error, code} body with the
// correct status and returns (nil, nil), which the wrapper treats as "already
// handled" and emits nothing further.
func ErrorMiddleware(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
	return func(c *fiber.Ctx, args any) (any, error) {
		resp, err := f(c, args)
		if err == nil {
			return resp, nil
		}

		status, code := fiber.StatusInternalServerError, "internal_error"
		switch {
		case errors.Is(err, repo.ErrNotFound):
			status, code = fiber.StatusNotFound, "not_found"
		case errors.Is(err, repo.ErrConflict):
			status, code = fiber.StatusConflict, "conflict"
		case errors.Is(err, repo.ErrBadCursor):
			status, code = fiber.StatusBadRequest, "bad_cursor"
		case errors.Is(err, repo.ErrBadRequest):
			status, code = fiber.StatusBadRequest, "bad_request"
		case errors.Is(err, overpass.ErrBadRequest):
			status, code = fiber.StatusBadRequest, "bad_request"
		case errors.Is(err, overpass.ErrRateLimited):
			status, code = fiber.StatusTooManyRequests, "rate_limited"
		case errors.Is(err, overpass.ErrTimeout):
			status, code = fiber.StatusGatewayTimeout, "overpass_timeout"
		case errors.Is(err, overpass.ErrUnavailable), errors.Is(err, overpass.ErrTooLarge):
			status, code = fiber.StatusBadGateway, "overpass_unavailable"
		}

		if writeErr := c.Status(status).JSON(api.Error{Error: err.Error(), Code: code}); writeErr != nil {
			return nil, writeErr
		}
		return nil, nil
	}
}
