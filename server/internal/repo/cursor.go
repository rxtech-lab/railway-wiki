package repo

import (
	"encoding/base64"
	"reflect"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	minLimit     = 1
)

// EncodeCursor turns a row id into an opaque forward cursor.
func EncodeCursor(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

// DecodeCursor reverses EncodeCursor.
func DecodeCursor(c string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// clampLimit bounds a requested limit into [minLimit, maxLimit], defaulting when nil.
func clampLimit(limit *int) int {
	if limit == nil {
		return defaultLimit
	}
	switch {
	case *limit < minLimit:
		return minLimit
	case *limit > maxLimit:
		return maxLimit
	default:
		return *limit
	}
}

// isNil reports whether v is a nil pointer/interface or an empty string, so that
// absent optional filters are skipped.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		return rv.IsNil()
	case reflect.String:
		return rv.String() == ""
	default:
		return false
	}
}

// deref unwraps a pointer to its element for use as a SQL bind value.
func deref(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		return rv.Elem().Interface()
	}
	return v
}
