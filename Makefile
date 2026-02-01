ifeq ($(OS),Windows_NT)
	DOCKER_COMPOSE = docker-compose
else
	ifeq ($(shell command -v docker-compose 2> /dev/null),)
		DOCKER_COMPOSE = docker compose
	else
		DOCKER_COMPOSE = docker-compose
	endif
endif

COMPOSE_DEV = -f docker-compose.dev.yml
COMPOSE_PROD = -f docker-compose.yml

.PHONY: clean help backend frontend format backend-test \
	develop-up develop-up-build develop-down develop-down-volumes develop-reup develop-rebuild \
	prod-up prod-up-build prod-down prod-down-volumes prod-reup prod-rebuild

develop-up:
	@echo "Starting containers (DEV - without Prometheus/Grafana)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) up

develop-up-build:
	@echo "Starting containers with build (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) up --build

develop-down:
	@echo "Stopping containers (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) down
	@echo "Done!"

develop-down-volumes:
	@echo "Stopping containers and removing volumes (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) down -v
	@echo "Done!"

develop-reup:
	@echo "Stopping and removing volumes (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) down -v
	@echo "Starting containers with build (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) up --build

develop-rebuild:
	@echo "Rebuilding images with no cache (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) build --no-cache
	@echo "Starting containers (DEV)..."
	$(DOCKER_COMPOSE) $(COMPOSE_DEV) up

prod-up:
	@echo "Starting containers (PROD - full stack)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) up

prod-up-build:
	@echo "Starting containers with build (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) up --build

prod-down:
	@echo "Stopping containers (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) down
	@echo "Done!"

prod-down-volumes:
	@echo "Stopping containers and removing volumes (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) down -v
	@echo "Done!"

prod-reup:
	@echo "Stopping and removing volumes (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) down -v
	@echo "Starting containers with build (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) up --build

prod-rebuild:
	@echo "Rebuilding images with no cache (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) build --no-cache
	@echo "Starting containers (PROD)..."
	$(DOCKER_COMPOSE) $(COMPOSE_PROD) up

clean:
	@echo "Stopping all containers..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "docker ps -a -q | ForEach-Object { if ($$_) { docker stop $$_ } }" 2>nul
else
	@docker ps -aq | xargs -r docker stop 2>/dev/null || true
endif
	@echo "Removing all containers..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "docker ps -a -q | ForEach-Object { if ($$_) { docker rm $$_ } }" 2>nul
else
	@docker ps -aq | xargs -r docker rm 2>/dev/null || true
endif
	@echo "Removing all images..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "docker images -q | ForEach-Object { if ($$_) { docker rmi -f $$_ } }" 2>nul
else
	@docker images -q | xargs -r docker rmi -f 2>/dev/null || true
endif
	@echo "Pruning Docker system..."
	docker system prune -a --volumes -f
	@echo "Docker cleanup complete!"

help:
	@echo "Available targets:"
	@echo ""
	@echo "DEV (docker-compose.dev.yml - db, auth, chat, frontend, proxy):"
	@echo "  develop-up           - Start containers (no build)"
	@echo "  develop-up-build     - Start containers with build"
	@echo "  develop-down         - Stop containers (keeps volumes)"
	@echo "  develop-down-volumes - Stop and remove volumes"
	@echo "  develop-reup         - Down -v + up --build"
	@echo "  develop-rebuild      - Build --no-cache + up"
	@echo ""
	@echo "PROD (docker-compose.yml - full stack + Prometheus, Grafana):"
	@echo "  prod-up           - Start containers (no build)"
	@echo "  prod-up-build     - Start containers with build"
	@echo "  prod-down         - Stop containers (keeps volumes)"
	@echo "  prod-down-volumes - Stop and remove volumes"
	@echo "  prod-reup         - Down -v + up --build"
	@echo "  prod-rebuild      - Build --no-cache + up"
	@echo ""
	@echo "UTILITIES:"
	@echo "  clean         - Remove ALL Docker containers, images, and volumes"
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
