// Package service provides a generic CRUD service that binds a generic
// repository to per-resource model<->API converters, returning API types so the
// HTTP handlers never touch database models.
package service

import (
	"context"

	"github.com/rxtech-lab/railway-wiki/internal/repo"
)

// Service is a generic CRUD service.
//
//	T  - GORM model, PT - *T
//	A  - API response type      (e.g. api.Company)
//	C  - API create-request type (e.g. api.CreateCompanyRequest)
//	U  - API update-request type (e.g. api.UpdateCompanyRequest)
type Service[T any, PT interface {
	*T
	repo.Model
}, A any, C any, U any] struct {
	repo    *repo.Repository[T, PT]
	toAPI   func(*T) A
	fromNew func(*C) T
	applyUp func(*U, *T)
}

// New constructs a Service from a repository and the three converter funcs.
func New[T any, PT interface {
	*T
	repo.Model
}, A any, C any, U any](
	r *repo.Repository[T, PT],
	toAPI func(*T) A,
	fromNew func(*C) T,
	applyUp func(*U, *T),
) *Service[T, PT, A, C, U] {
	return &Service[T, PT, A, C, U]{repo: r, toAPI: toAPI, fromNew: fromNew, applyUp: applyUp}
}

// List returns a page of API objects plus the next cursor.
func (s *Service[T, PT, A, C, U]) List(ctx context.Context, f repo.Filter) ([]A, *string, error) {
	rows, next, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	out := make([]A, len(rows))
	for i := range rows {
		out[i] = s.toAPI(&rows[i])
	}
	return out, next, nil
}

// Get returns a single API object.
func (s *Service[T, PT, A, C, U]) Get(ctx context.Context, id string) (*A, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	a := s.toAPI(m)
	return &a, nil
}

// Create inserts a new row from a create request and returns the API object.
func (s *Service[T, PT, A, C, U]) Create(ctx context.Context, body *C) (*A, error) {
	m := s.fromNew(body)
	if err := s.repo.Create(ctx, &m); err != nil {
		return nil, err
	}
	a := s.toAPI(&m)
	return &a, nil
}

// Update applies an update request to an existing row and returns the API object.
func (s *Service[T, PT, A, C, U]) Update(ctx context.Context, id string, body *U) (*A, error) {
	m, err := s.repo.Update(ctx, id, func(m *T) { s.applyUp(body, m) })
	if err != nil {
		return nil, err
	}
	a := s.toAPI(m)
	return &a, nil
}

// Delete removes a row by id.
func (s *Service[T, PT, A, C, U]) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
