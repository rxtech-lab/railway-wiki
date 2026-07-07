package dto

// CreateExampleDTO represents the request body for creating an example
type CreateExampleDTO struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// UpdateExampleDTO represents the request body for updating an example
type UpdateExampleDTO struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}
