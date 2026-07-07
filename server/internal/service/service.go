package service

import (
	"context"
	"time"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExampleService handles business logic for examples
type ExampleService struct {
	db *gorm.DB
}

// NewExampleService creates a new ExampleService
func NewExampleService(db *gorm.DB) *ExampleService {
	return &ExampleService{db: db}
}

// List returns all examples
func (s *ExampleService) List(ctx context.Context) ([]api.Example, error) {
	var examples []models.Example
	if err := s.db.WithContext(ctx).Find(&examples).Error; err != nil {
		return nil, err
	}

	result := make([]api.Example, len(examples))
	for i, e := range examples {
		result[i] = toAPIExample(e)
	}
	return result, nil
}

// GetByID returns a single example by ID
func (s *ExampleService) GetByID(ctx context.Context, id string) (*api.Example, error) {
	var example models.Example
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&example).Error; err != nil {
		return nil, err
	}
	result := toAPIExample(example)
	return &result, nil
}

// Create creates a new example
func (s *ExampleService) Create(ctx context.Context, name string, description *string) (*api.Example, error) {
	example := models.Example{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(&example).Error; err != nil {
		return nil, err
	}

	result := toAPIExample(example)
	return &result, nil
}

// toAPIExample converts a model to API response
func toAPIExample(e models.Example) api.Example {
	return api.Example{
		Id:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   &e.UpdatedAt,
	}
}
