package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/auth"
	"github.com/rxtech-lab/railway-wiki/internal/media"
	"github.com/rxtech-lab/railway-wiki/internal/overpass"
	"github.com/rxtech-lab/railway-wiki/internal/schema"
	"github.com/rxtech-lab/railway-wiki/internal/server"
	"github.com/rxtech-lab/railway-wiki/internal/testutil"
)

const testToken = "test-token"

// testAuthenticator is a minimal stand-in for the OAuth/JWKS authenticator: it
// grants the admin role to a single known token so the suite can exercise the
// management middleware without a real identity provider.
type testAuthenticator struct{}

type testOverpass struct{}

func (testOverpass) Search(context.Context, overpass.SearchRequest) ([]overpass.Candidate, error) {
	return nil, overpass.ErrUnavailable
}

func (testOverpass) Fetch(context.Context, string, int64) (*overpass.Candidate, error) {
	return nil, overpass.ErrUnavailable
}

func (testAuthenticator) Authenticate(_ context.Context, token string) (*auth.Principal, error) {
	if token != testToken {
		return nil, auth.ErrUnauthorized
	}
	return &auth.Principal{Subject: "test", Roles: []string{auth.AdminRole}}, nil
}

type ServerTestSuite struct {
	testutil.DBTestSuite
	app *fiber.App
}

func (s *ServerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()

	reg, err := schema.NewRegistry()
	s.Require().NoError(err)

	srv := server.NewServer(s.DB, media.NoopPresigner{}, reg, testOverpass{})

	app := fiber.New()
	app.Use("/api/management", auth.RequireRole(testAuthenticator{}, auth.AdminRole))
	strict := api.NewStrictHandler(srv, []api.StrictMiddlewareFunc{server.ErrorMiddleware})
	api.RegisterHandlers(app, strict)
	s.app = app
}

// --- helpers ---

func (s *ServerTestSuite) do(method, path string, body any, token string) *http.Response {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		s.Require().NoError(err)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)
	return resp
}

func decode[T any](s *ServerTestSuite, resp *http.Response) T {
	var out T
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&out))
	return out
}

func (s *ServerTestSuite) createCompany(name string) api.Company {
	resp := s.do(http.MethodPost, "/api/management/companies",
		api.CreateCompanyRequest{Name: name}, testToken)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	return decode[api.Company](s, resp)
}

// --- tests ---

func (s *ServerTestSuite) TestHealth() {
	resp := s.do(http.MethodGet, "/health", nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	body := decode[api.HealthResponse](s, resp)
	s.Equal("ok", body.Status)
}

func (s *ServerTestSuite) TestCreateGetCompany() {
	created := s.createCompany("Central Railway")
	s.Require().NotNil(created.Id)
	s.Equal("Central Railway", created.Name)

	// public get returns the bare object
	resp := s.do(http.MethodGet, "/api/companies/"+created.Id.String(), nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	got := decode[api.Company](s, resp)
	s.Equal(created.Id.String(), got.Id.String())
	s.Equal("Central Railway", got.Name)
}

func (s *ServerTestSuite) TestGetUnknownReturns404() {
	resp := s.do(http.MethodGet, "/api/companies/00000000-0000-0000-0000-000000000000", nil, "")
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	e := decode[api.Error](s, resp)
	s.Equal("not_found", e.Code)
}

func (s *ServerTestSuite) TestManagementRequiresAuth() {
	// missing token -> 401
	resp := s.do(http.MethodPost, "/api/management/companies", api.CreateCompanyRequest{Name: "X"}, "")
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)

	// bad token -> 401
	resp = s.do(http.MethodPost, "/api/management/companies", api.CreateCompanyRequest{Name: "X"}, "wrong")
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *ServerTestSuite) TestPublicListIsOpen() {
	resp := s.do(http.MethodGet, "/api/companies", nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func (s *ServerTestSuite) TestCursorPagination() {
	for _, n := range []string{"A", "B", "C"} {
		s.createCompany(n)
	}

	// first page of 2
	resp := s.do(http.MethodGet, "/api/companies?limit=2", nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	page := decode[api.CompanyPage](s, resp)
	s.Require().Len(page.Items, 2)
	s.Require().NotNil(page.Pagination.Next)

	// follow cursor -> remaining 1
	resp = s.do(http.MethodGet, "/api/companies?limit=2&cursor="+*page.Pagination.Next, nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	page2 := decode[api.CompanyPage](s, resp)
	s.Require().Len(page2.Items, 1)
	s.Nil(page2.Pagination.Next)
}

func (s *ServerTestSuite) TestFilterByForeignKey() {
	c := s.createCompany("Owner")
	// two routes, one linked to the company via route-companies
	route := s.do(http.MethodPost, "/api/management/routes", api.CreateRouteRequest{Name: "Line 1"}, testToken)
	s.Require().Equal(http.StatusCreated, route.StatusCode)
	r := decode[api.Route](s, route)

	rc := s.do(http.MethodPost, "/api/management/route-companies",
		api.CreateRouteCompanyRequest{RouteId: *r.Id, CompanyId: *c.Id}, testToken)
	s.Require().Equal(http.StatusCreated, rc.StatusCode)

	resp := s.do(http.MethodGet, "/api/route-companies?companyId="+c.Id.String(), nil, "")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	page := decode[api.RouteCompanyPage](s, resp)
	s.Require().Len(page.Items, 1)
	s.Equal(r.Id.String(), page.Items[0].RouteId.String())
}

func (s *ServerTestSuite) TestUpdateAndDelete() {
	created := s.createCompany("Old Name")

	upd := s.do(http.MethodPut, "/api/management/companies/"+created.Id.String(),
		api.UpdateCompanyRequest{Name: "New Name"}, testToken)
	s.Require().Equal(http.StatusOK, upd.StatusCode)
	s.Equal("New Name", decode[api.Company](s, upd).Name)

	del := s.do(http.MethodDelete, "/api/management/companies/"+created.Id.String(), nil, testToken)
	s.Require().Equal(http.StatusNoContent, del.StatusCode)

	get := s.do(http.MethodGet, "/api/companies/"+created.Id.String(), nil, "")
	s.Require().Equal(http.StatusNotFound, get.StatusCode)
}

func (s *ServerTestSuite) TestSchemaEndpoint() {
	resp := s.do(http.MethodGet, "/api/management/companies/schema?action=create", nil, testToken)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	doc := decode[map[string]any](s, resp)
	s.Equal("https://json-schema.org/draft/2020-12/schema", doc["$schema"])
	s.Contains(doc, "properties")
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
