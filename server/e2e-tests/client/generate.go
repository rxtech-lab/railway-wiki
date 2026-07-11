// Package client holds the generated typed HTTP client used by the e2e suite.
//
// The client is generated from api/openapi.yaml by oapi-codegen. Run
// `make e2e-client` (or `go generate ./e2e-tests/client/...`) to regenerate
// client.gen.go after changing the OpenAPI spec.
package client

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml ../../api/openapi.yaml
