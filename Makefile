# Makefile for db-go

# Variables
DC := docker compose

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make up        - Start all containers (PostgreSQL and Datadog agent)"
	@echo "  make down      - Stop and remove all containers"
	@echo "  make restart   - Restart all containers"
	@echo "  make logs      - Show logs of all containers"
	@echo "  make ps        - Show status of containers"
	@echo "  make pg-shell  - Connect to PostgreSQL shell"
	@echo "  make example   - Run the datadog example"

# Start containers
.PHONY: up
up:
	$(DC) up -d

# Stop containers
.PHONY: down
down:
	$(DC) down

# Restart containers
.PHONY: restart
restart:
	$(DC) down
	$(DC) up -d

# Show logs
.PHONY: logs
logs:
	$(DC) logs -f

# Show status
.PHONY: ps
ps:
	$(DC) ps

# Connect to PostgreSQL shell
.PHONY: pg-shell
pg-shell:
	$(DC) exec postgres psql -U postgres -d example

# Run the datadog example
.PHONY: example
example:
	cd example/datadog && go run main.go
