# Makefile for db-go

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build     - Build the package"
	@echo "  make test      - Run tests"
	@echo "  make test-v    - Run tests with verbose output"
	@echo "  make vet       - Run go vet"
	@echo "  make fmt       - Run gofmt (check only)"
	@echo "  make fmt-fix   - Run gofmt and fix files"
	@echo "  make lint      - Run vet + fmt check"
	@echo "  make up        - Start all containers (PostgreSQL and Datadog agent)"
	@echo "  make down      - Stop and remove all containers"
	@echo "  make restart   - Restart all containers"
	@echo "  make logs      - Show logs of all containers"
	@echo "  make ps        - Show status of containers"
	@echo "  make pg-shell  - Connect to PostgreSQL shell"
	@echo "  make example   - Run the datadog example"
	@echo "  make example-usecase - Run the usecase example"

# Build
.PHONY: build
build:
	go mod tidy
	go build ./...

# Run tests
.PHONY: test
test:
	go test ./... -count=1

# Run tests (verbose)
.PHONY: test-v
test-v:
	go test ./... -v -count=1

# Run go vet
.PHONY: vet
vet:
	go vet ./...

# Check formatting (no write)
.PHONY: fmt
fmt:
	@test -z "$$(go fmt ./...)" || (echo "Run 'make fmt-fix' to fix formatting"; exit 1)

# Fix formatting
.PHONY: fmt-fix
fmt-fix:
	go fmt ./...

# Lint: vet + fmt check
.PHONY: lint
lint: vet fmt

# Start containers
.PHONY: up
up:
	docker compose up -d

# Stop containers
.PHONY: down
down:
	docker compose down

# Restart containers
.PHONY: restart
restart:
	docker compose down
	docker compose up -d

# Show logs
.PHONY: logs
logs:
	docker compose logs -f

# Show status
.PHONY: ps
ps:
	docker compose ps

# Connect to PostgreSQL shell
.PHONY: pg-shell
pg-shell:
	docker compose exec postgres psql -U postgres -d example

# Run the datadog example
.PHONY: example
example:
	cd example/datadog && go run main.go

# Run the usecase example
.PHONY: example-usecase
example-usecase:
	cd example/usecase && go run main.go
