package server

import (
	"context"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/service"
)

// Ensure Server implements StrictServerInterface
var _ api.StrictServerInterface = (*Server)(nil)

// Server implements api.StrictServerInterface
type Server struct {
	exampleService *service.ExampleService
}

// NewServer creates a new server instance (used by Wire)
func NewServer(exampleService *service.ExampleService) *Server {
	return &Server{
		exampleService: exampleService,
	}
}

// GetHealth implements api.StrictServerInterface
func (s *Server) GetHealth(ctx context.Context, request api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{
		Status: "ok",
	}, nil
}

// ListExamples implements api.StrictServerInterface
func (s *Server) ListExamples(ctx context.Context, request api.ListExamplesRequestObject) (api.ListExamplesResponseObject, error) {
	examples, err := s.exampleService.List(ctx)
	if err != nil {
		return nil, err
	}
	return api.ListExamples200JSONResponse(examples), nil
}

// CreateExample implements api.StrictServerInterface
func (s *Server) CreateExample(ctx context.Context, request api.CreateExampleRequestObject) (api.CreateExampleResponseObject, error) {
	example, err := s.exampleService.Create(ctx, request.Body.Name, request.Body.Description)
	if err != nil {
		return api.CreateExample400JSONResponse{
			Message: err.Error(),
		}, nil
	}
	return api.CreateExample201JSONResponse(*example), nil
}

// GetExample implements api.StrictServerInterface
func (s *Server) GetExample(ctx context.Context, request api.GetExampleRequestObject) (api.GetExampleResponseObject, error) {
	example, err := s.exampleService.GetByID(ctx, request.Id.String())
	if err != nil {
		return api.GetExample404JSONResponse{
			Message: "Example not found",
		}, nil
	}
	return api.GetExample200JSONResponse(*example), nil
}
