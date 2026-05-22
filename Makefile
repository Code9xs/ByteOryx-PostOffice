.PHONY: build run test clean migrate-up migrate-down docker-up docker-down

BINARY=postoffice
BUILD_DIR=./bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/postoffice

run:
	go run ./cmd/postoffice -config configs/postoffice.yaml

test:
	go test ./... -v

clean:
	rm -rf $(BUILD_DIR)

migrate-up:
	migrate -path internal/infrastructure/postgres/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/infrastructure/postgres/migrations -database "$(DATABASE_URL)" down

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

docker-build:
	docker compose -f deployments/docker-compose.yml build

lint:
	golangci-lint run ./...

deps:
	go mod tidy
