DOCKER_COMPOSE ?= docker compose

COMPOSE_DEV = -f docker-compose.dev.yml
COMPOSE_PROD = -f docker-compose.yml

.PHONY: up down prod-up prod-down clean help backend frontend format backend-test

up:
	@echo "Starting containers with build (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) up --build

down:
	@echo "Stopping containers (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) down
	@echo "Done!"

prod-up:
	@echo "Starting containers with build (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) up --build

prod-down:
	@echo "Stopping containers (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) down
	@echo "Done!"

clean:
	@echo "Removing this project's containers, local images, and volumes..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) down -v --rmi local --remove-orphans
	@echo "Project cleanup complete!"

help:
	@echo "Available targets:"
	@echo ""
	@echo "  up           - Build and start the development stack"
	@echo "  down         - Stop the development stack"
	@echo "  prod-up      - Build and start the full production stack"
	@echo "  prod-down    - Stop the production stack"
	@echo "  clean         - Remove this project's containers, local images, and volumes"
	@echo "  backend      - Run backend services locally without Docker"
	@echo "  frontend     - Run frontend locally without Docker"
	@echo "  format       - Format and lint all code (backend + frontend)"
	@echo "  backend-test - Run all backend tests"

backend:
	cd backend && go run ./cmd/auth &
	cd backend && go run ./cmd/chat &

frontend:
	cd frontend && npm run dev

format:
	@echo "=== Backend ==="
	@echo "Running go fmt..."
	cd backend && goimports -w .
	@echo "Running go vet..."
	cd backend && go vet ./...
	@echo "Running go lint..."
	cd backend && golangci-lint run ./...
	@echo ""
	@echo "=== Frontend ==="
	@echo "Running TypeScript type check..."
	cd frontend && npm run type-check
	@echo "Formatting code with Prettier..."
	cd frontend && npm run format
	@echo "Running ESLint with auto-fix..."
	cd frontend && npm run lint:fix
	@echo ""
	@echo "All code formatted and linted!"

backend-test:
	@echo "=== Backend: all tests ==="
	cd backend && go test -count=1 ./test/...
	@echo ""
	@echo "Backend tests OK"
