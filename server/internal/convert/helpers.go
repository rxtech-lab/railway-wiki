// Package convert holds pure model<->API mapping functions, the only
// per-resource code besides the (mechanical) handlers.
package convert

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/datatypes"
)

// --- UUID helpers (models store ids as canonical strings) ---

func idPtr(s string) *openapi_types.UUID {
	u, _ := uuid.Parse(s)
	return &u
}

func toUUID(s string) openapi_types.UUID {
	u, _ := uuid.Parse(s)
	return u
}

func toUUIDPtr(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	u, _ := uuid.Parse(*s)
	return &u
}

func uuidStr(u openapi_types.UUID) string {
	return u.String()
}

func uuidStrPtr(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

// --- date / date-time helpers ---

func datePtr(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func date(t time.Time) openapi_types.Date {
	return openapi_types.Date{Time: t}
}

func fromDatePtr(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

func fromDate(d openapi_types.Date) time.Time {
	return d.Time
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// --- GeoJSON helpers (stored as JSON text) ---

func toGeo(j datatypes.JSON) *map[string]interface{} {
	if len(j) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(j, &m); err != nil {
		return nil
	}
	return &m
}

func fromGeo(m *map[string]interface{}) datatypes.JSON {
	if m == nil {
		return nil
	}
	b, err := json.Marshal(*m)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}
