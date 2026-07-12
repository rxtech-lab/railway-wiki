package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/rxtech-lab/railway-wiki/internal/api"
	"github.com/rxtech-lab/railway-wiki/internal/auth"
	"github.com/rxtech-lab/railway-wiki/internal/config"
	"github.com/rxtech-lab/railway-wiki/internal/database"
	"github.com/rxtech-lab/railway-wiki/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	} else {
		log.Println("Loaded configuration from .env file")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Build the management authenticator (used as prefix middleware below).
	// Skipped entirely in E2E mode, where management auth is disabled.
	var authenticator auth.Authenticator
	if !cfg.E2EMode {
		authenticator, err = server.ProvideAuthenticator(cfg)
		if err != nil {
			log.Fatalf("Failed to initialize authenticator: %v", err)
		}
	}

	// Wire the strict server (services + schema registry + presigner).
	strictServer, err := server.InitializeServer(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if cfg.E2EMode {
		log.Println("⚠️  E2E_MODE enabled: management API auth is DISABLED — do not use in production")
		if err := database.Seed(db); err != nil {
			log.Fatalf("Failed to seed e2e data: %v", err)
		}
	}

	app := fiber.New()
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Enforce the `admin` role on every management endpoint. Registered before
	// the strict routes so it runs first for the /api/management prefix. Skipped
	// in E2E mode so UI tests can drive the API without an OAuth flow.
	if !cfg.E2EMode {
		app.Use("/api/management", auth.RequireRole(authenticator, auth.AdminRole))
	}

	// Register handlers. ErrorMiddleware maps sentinel errors to HTTP statuses.
	strictHandler := api.NewStrictHandler(strictServer, []api.StrictMiddlewareFunc{server.ErrorMiddleware})
	api.RegisterHandlers(app, strictHandler)

	log.Printf("Starting server on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
