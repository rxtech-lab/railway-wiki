//go:build wireinject
// +build wireinject

package server

import (
	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/service"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// InitializeServer creates a new server with all dependencies injected
func InitializeServer(db *gorm.DB) (api.StrictServerInterface, error) {
	wire.Build(
		service.NewExampleService,
		NewServer,
		wire.Bind(new(api.StrictServerInterface), new(*Server)),
	)
	return nil, nil
}
