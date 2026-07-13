# railway-wiki

Go server built with Fiber, GORM, and Wire, backed by **TursoDB (libSQL)**.

The API (see `api/openapi.yaml`) exposes public read-only endpoints under
`/api/{resource}` and bearer-protected management CRUD under
`/api/management/{resource}` for 22 railway domain resources (companies,
stations, routes, trains, timetables, media, …). Management access requires role
`admin`. List endpoints use opaque
cursor pagination (`{ items, pagination: { next } }`); errors are `{ error, code }`.

## Getting Started

### Prerequisites

- Go 1.24 or higher
- A TursoDB/libSQL database (or a local SQLite file for development)
- Docker (optional, for localstack S3 when testing media uploads)

### Environment Setup

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Update the `.env` file with your configuration (see `.env.example` for all
   options — database, management auth, and S3 media storage):
```env
# Local SQLite file (dev) or a remote Turso URL:
#   DATABASE_URL=libsql://<db>-<org>.turso.io?authToken=<token>
DATABASE_URL=file:railway.db
PORT=8080

# Management endpoints require an OAuth/OIDC JWT access token whose `roles` array includes `admin`.
# The JWKS is discovered from the issuer via OIDC discovery:
OAUTH_ISSUER=https://issuer.example.com/
```

### Installation

1. Install dependencies:
```bash
make install
```

2. Generate code:
```bash
make generate
```

### Running the Server

#### Using Docker Compose (Recommended)

Start the database and server:
```bash
docker compose up
```

#### Local Development

1. Start PostgreSQL (if not using Docker)

2. Run the server:
```bash
make run
```

The server will start on `http://localhost:8080` (or the port specified in your `.env` file).

### Building

Build the server binary:
```bash
make build
```

The binary will be created in `bin/server`.

### Testing

Run tests:
```bash
make test
```

### API Documentation

The API is documented using OpenAPI 3.0. View the specification at:
- File: `api/openapi.yaml`
- Swagger UI: `http://localhost:8080/swagger` (when server is running)

### Available Endpoints

- `GET /health` - Health check
- `GET /api/{resource}` - Public, cursor-paginated list (supports `cursor`, `limit`, filters, `q`)
- `GET /api/{resource}/{id}` - Public get by id
- `GET /api/management/{resource}` - Management list (requires an admin bearer token)
- `POST /api/management/{resource}` - Create
- `GET /api/management/{resource}/{id}` - Get by id
- `PUT /api/management/{resource}/{id}` - Replace
- `DELETE /api/management/{resource}/{id}` - Delete
- `GET /api/management/{resource}/schema?action=create|update` - JSON Schema for the request body
- `POST /api/management/media/upload-url` - Presigned media upload target
- `GET /api/management/dashboard` - Counts and station coordinate coverage
- `GET /api/management/overpass/stations` - Bounded railway candidate search
- `POST /api/management/overpass/stations/{type}/{id}/import` - Verified OSM import
- `GET|PUT /api/management/routes/{id}/configuration` - Atomic route editor data
- `POST /api/management/routes/{id}/configuration/validate` - Validate an unsaved route

Resources: `companies`, `stations`, `station-codes`, `station-transfers`,
`platforms`, `routes`, `route-companies`, `route-stations`, `track-segments`,
`platform-tracks`, `operation-routes`, `operation-route-companies`,
`operation-route-sections`, `operation-route-stops`, `timetable-versions`,
`service-calendars`, `service-calendar-exceptions`, `trains`, `train-runs`,
`train-run-stops`, `media`, `media-attachments`.

The app never sends arbitrary Overpass QL. It requests a bounded viewport or
point search from the management API; the server builds the query, applies
timeouts/response limits/cache/throttling, and defaults to the public
`https://overpass-api.de/api/interpreter` endpoint. A private endpoint and its
server-only key can still be configured with `OVERPASS_UPSTREAM_URL` and
`OVERPASS_API_KEY`.

### Project Structure

```
.
├── api/                  # OpenAPI specifications
├── cmd/                  # Application entry points
│   └── server/          # Main server application
├── internal/            # Private application code
│   ├── api/            # Generated API code
│   ├── config/         # Configuration management
│   ├── database/       # Database connections and migrations
│   ├── dto/            # Data transfer objects
│   ├── models/         # Database models
│   ├── server/         # HTTP server implementation
│   ├── service/        # Business logic
│   ├── testutil/       # Testing utilities
│   └── utils/          # Utility functions
├── k8s/                 # Kubernetes manifests
└── tools/               # Build tools and generators
```

### Development

#### Adding a New Endpoint

1. Update the OpenAPI specification in `api/openapi.yaml`
2. Run `make generate` to regenerate the API code
3. Implement the endpoint in `internal/server/server.go`
4. Add business logic in `internal/service/`

#### Database Migrations

Migrations are automatically run on server startup using GORM AutoMigrate.
Add new models in `internal/models/` and update the `Migrate` function in `internal/database/database.go`.

## Deployment



### Docker

Build and tag the Docker image:
```bash
make docker
```

## License

Add your license information here.
