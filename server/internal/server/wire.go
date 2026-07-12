//go:build wireinject
// +build wireinject

package server

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"github.com/rxtech-lab/railway-wiki/internal/config"
	"github.com/rxtech-lab/railway-wiki/internal/schema"
)

// InitializeServer wires the strict server together from config + db.
// The management authenticator is built separately (server.ProvideAuthenticator)
// because it is used as Fiber middleware rather than injected into the Server.
func InitializeServer(cfg *config.Config, db *gorm.DB) (*Server, error) {
	wire.Build(
		ProvidePresigner,
		ProvideOverpassClient,
		schema.NewRegistry,
		NewServer,
	)
	return nil, nil
}
