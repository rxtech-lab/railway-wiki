.PHONY: install build test generate run docker clean

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
	go mod tidy
	go tool github.com/google/wire/cmd/wire ./internal/server

run:
	go run cmd/server/main.go


docker:
	docker build -t railway-wiki:latest .
	docker tag railway-wiki:latest railway-wiki:$$(git describe --tags --always 2>/dev/null || echo "dev")




clean:
	rm -rf bin/
	rm -f internal/api/gen.go
