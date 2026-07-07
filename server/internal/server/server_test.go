package server

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	app *fiber.App
}

func (s *ServerTestSuite) SetupTest() {
	s.app = fiber.New()
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}

func (s *ServerTestSuite) TestHealthEndpoint() {
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := s.app.Test(req)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(s.T(), string(body), "ok")
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
