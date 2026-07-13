// Package repo provides a generic GORM-backed repository with cursor pagination
// and simple equality + free-text filtering, shared by every resource.
package repo

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Sentinel errors mapped to HTTP statuses by the server error middleware.
var (
	ErrNotFound   = errors.New("resource not found")
	ErrConflict   = errors.New("resource conflict")
	ErrBadCursor  = errors.New("invalid cursor")
	ErrBadRequest = errors.New("invalid request")
)

// Model is satisfied by every model via the embedded models.Base.
type Model interface {
	GetID() string
}

// Filter describes a list query.
type Filter struct {
	// Eq holds column => value equality constraints. Entries whose value is a
	// nil pointer or empty are skipped, so absent query params are no-ops.
	Eq map[string]any
	// Q is an optional free-text term matched (LIKE) across the repo's searchCols.
	Q *string
	// Cursor is an opaque pagination cursor from a previous page.
	Cursor *string
	// Limit is the requested page size (nil => default).
	Limit *int
	// Optional station viewport. Either all four values are supplied or none.
	South *float64
	West  *float64
	North *float64
	East  *float64
}

// Repository is a generic CRUD repository for model T (pointer type PT).
type Repository[T any, PT interface {
	*T
	Model
}] struct {
	db         *gorm.DB
	searchCols []string
}

// New builds a repository. searchCols are the columns scanned by Filter.Q.
func New[T any, PT interface {
	*T
	Model
}](db *gorm.DB, searchCols ...string) *Repository[T, PT] {
	return &Repository[T, PT]{db: db, searchCols: searchCols}
}

// List returns a page of rows ordered by id plus the next-page cursor (nil on
// the last page). Pagination is keyset: WHERE id > cursor ORDER BY id LIMIT n+1.
func (r *Repository[T, PT]) List(ctx context.Context, f Filter) ([]T, *string, error) {
	q := r.db.WithContext(ctx).Model(new(T))

	hasBounds := f.South != nil || f.West != nil || f.North != nil || f.East != nil
	if hasBounds {
		if f.South == nil || f.West == nil || f.North == nil || f.East == nil ||
			*f.South < -90 || *f.North > 90 || *f.West < -180 || *f.East > 180 ||
			*f.South >= *f.North || *f.West >= *f.East {
			return nil, nil, ErrBadRequest
		}
		q = q.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?", *f.South, *f.North, *f.West, *f.East)
	}

	for col, v := range f.Eq {
		if isNil(v) {
			continue
		}
		q = q.Where(col+" = ?", deref(v))
	}

	if f.Q != nil && *f.Q != "" && len(r.searchCols) > 0 {
		like := "%" + *f.Q + "%"
		sub := r.db
		for i, c := range r.searchCols {
			if i == 0 {
				sub = sub.Where(c+" LIKE ?", like)
			} else {
				sub = sub.Or(c+" LIKE ?", like)
			}
		}
		q = q.Where(sub)
	}

	if f.Cursor != nil && *f.Cursor != "" {
		id, err := DecodeCursor(*f.Cursor)
		if err != nil {
			return nil, nil, ErrBadCursor
		}
		q = q.Where("id > ?", id)
	}

	limit := clampLimit(f.Limit)

	var items []T
	if err := q.Order("id ASC").Limit(limit + 1).Find(&items).Error; err != nil {
		return nil, nil, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		last := PT(&items[limit-1]).GetID()
		c := EncodeCursor(last)
		next = &c
	}
	return items, next, nil
}

// Get returns a single row by id, or ErrNotFound.
func (r *Repository[T, PT]) Get(ctx context.Context, id string) (*T, error) {
	var m T
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// Create inserts a new row (id is assigned by the model's BeforeCreate hook).
func (r *Repository[T, PT]) Create(ctx context.Context, m *T) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return classify(err)
	}
	return nil
}

// Update loads a row, applies the mutation, and saves it. Returns ErrNotFound
// when the row does not exist.
func (r *Repository[T, PT]) Update(ctx context.Context, id string, apply func(*T)) (*T, error) {
	var m T
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	apply(&m)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, classify(err)
	}
	return &m, nil
}

// Delete removes a row by id, or returns ErrNotFound.
func (r *Repository[T, PT]) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(new(T))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// classify maps driver errors to sentinels.
func classify(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed") {
		return ErrConflict
	}
	return err
}
