.PHONY: install build test generate run run\:no-check docker clean e2e-client test-e2e

install:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/google/wire/cmd/wire@latest

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./...

generate:
	go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get github.com/google/wire/cmd/wire
	go generate ./tools/...
	cp api/openapi.yaml internal/schema/openapi.yaml
	python3 scripts/gen_handlers.py
	gofmt -w internal/server/server.go
	go mod tidy
	go tool github.com/google/wire/cmd/wire ./internal/server

run:
	go run cmd/server/main.go

# Run against the configured development database without management role checks.
run\:no-check:
	SKIP_ROLE_CHECK=true go run cmd/server/main.go

# Run the server in E2E mode: local SQLite, management auth DISABLED, sample data
# seeded on startup. Used for iOS UI tests and manual QA against a real server.
run-e2e:
	E2E_MODE=true DATABASE_URL=file:$(CURDIR)/tmp/ios-e2e.db PORT=8080 go run cmd/server/main.go

# Regenerate the e2e HTTP client from api/openapi.yaml. Run after changing the spec.
e2e-client:
	go generate ./e2e-tests/client/...

# Run the black-box e2e suite against a local file SQLite DB + dockerized MinIO.
# Boots MinIO via docker compose, waits for it, then runs the e2e-tagged tests
# (the server binary is built and launched as a subprocess by the test harness).
E2E_ENV = \
	DATABASE_URL=file:$(CURDIR)/tmp/railway-e2e.db \
	S3_ENDPOINT=http://localhost:9000 \
	S3_BUCKET=railway-wiki \
	S3_REGION=us-east-1 \
	S3_ACCESS_KEY_ID=minioadmin \
	S3_SECRET_ACCESS_KEY=minioadmin \
	S3_PATH_STYLE=true \
	S3_PUBLIC_URL=http://localhost:9000/railway-wiki \
	S3_PRESIGN_TTL=15m

test-e2e: e2e-client
	docker compose up -d minio createbuckets
	@echo "Waiting for MinIO to be ready..."
	@timeout=60; until curl -sf http://localhost:9000/minio/health/ready >/dev/null 2>&1 || [ $$timeout -le 0 ]; do sleep 1; timeout=$$((timeout-1)); done
	$(E2E_ENV) go test -tags e2e -count=1 -v ./e2e-tests/...


docker:
	docker build -f Dockerfile.vercel -t railway-wiki:latest .
	docker tag railway-wiki:latest railway-wiki:$$(git describe --tags --always 2>/dev/null || echo "dev")




clean:
	rm -rf bin/
	rm -f internal/api/gen.go
