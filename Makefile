ifeq ($(OS),Windows_NT)
	DOCKER_COMPOSE = docker-compose
else
	ifeq ($(shell command -v docker-compose 2> /dev/null),)
		DOCKER_COMPOSE = docker compose
	else
		DOCKER_COMPOSE = docker-compose
	endif
endif

COMPOSE_FILES_DEVELOP = -f docker-compose.yml \
	-f yaml/db.yml \
	-f yaml/auth.yml \
	-f yaml/chat.yml \
	-f yaml/frontend.yml \
	-f yaml/proxy.yml

COMPOSE_FILES_PROD = -f docker-compose.yml \
	-f yaml/db.yml \
	-f yaml/auth.yml \
	-f yaml/chat.yml \
	-f yaml/frontend.yml \
	-f yaml/proxy.yml \
	-f yaml/prometheus.yml \
	-f yaml/grafana.yml

.PHONY: clean help backend frontend go-fmt go-vet go-test format develop-up develop-up-build develop-down develop-down-volumes develop-restart develop-reup develop-rebuild prod-up prod-up-build prod-down prod-down-volumes prod-restart prod-reup prod-rebuild

develop-up:
	@echo "Starting containers (DEVELOP mode - minimal)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) up

develop-up-build:
	@echo "Starting containers with build (DEVELOP mode - minimal)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) up --build

develop-down:
	@echo "Stopping containers (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) down
	@echo "Done!"

develop-down-volumes:
	@echo "Stopping containers and removing volumes (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) down -v
	@echo "Done!"

develop-restart:
	@echo "Restarting containers (DEVELOP mode - keeps volumes and cache)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) restart
	@echo "Done!"

develop-reup:
	@echo "Stopping containers and removing volumes (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) down -v
	@echo "Starting containers with build (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) up --build

develop-rebuild:
	@echo "Rebuilding images with no cache (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) build --no-cache
	@echo "Starting containers (DEVELOP mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_DEVELOP) up

prod-up:
	@echo "Starting containers (PROD mode - full stack)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) up

prod-up-build:
	@echo "Starting containers with build (PROD mode - full stack)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) up --build

prod-down:
	@echo "Stopping containers (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) down
	@echo "Done!"

prod-down-volumes:
	@echo "Stopping containers and removing volumes (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) down -v
	@echo "Done!"

prod-restart:
	@echo "Restarting containers (PROD mode - keeps volumes and cache)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) restart
	@echo "Done!"

prod-reup:
	@echo "Stopping containers and removing volumes (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) down -v
	@echo "Starting containers with build (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) up --build

prod-rebuild:
	@echo "Rebuilding images with no cache (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) build --no-cache
	@echo "Starting containers (PROD mode)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES_PROD) up

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
	@echo "DEVELOP MODE (minimal - without Prometheus/Grafana):"
	@echo "  develop-up           - Start containers (fast, no build)"
	@echo "  develop-up-build     - Start containers with build"
	@echo "  develop-down         - Stop containers (keeps volumes)"
	@echo "  develop-down-volumes - Stop containers and remove volumes"
	@echo "  develop-restart      - Restart containers (fastest, keeps everything)"
	@echo "  develop-reup         - Stop + rebuild + start"
	@echo "  develop-rebuild      - Full rebuild without cache"
	@echo ""
	@echo "PROD MODE (full - with Prometheus/Grafana):"
	@echo "  prod-up           - Start containers (fast, no build)"
	@echo "  prod-up-build     - Start containers with build"
	@echo "  prod-down         - Stop containers (keeps volumes)"
	@echo "  prod-down-volumes - Stop containers and remove volumes"
	@echo "  prod-restart      - Restart containers (fastest, keeps everything)"
	@echo "  prod-reup         - Stop + rebuild + start"
	@echo "  prod-rebuild      - Full rebuild without cache"
	@echo ""
	@echo "UTILITIES:"
	@echo "  clean        - Remove ALL Docker containers, images, and volumes"
	@echo "  backend      - Run backend services locally without Docker"
	@echo "  frontend     - Run frontend locally without Docker"
	@echo "  go-fmt       - Run go fmt for all backend packages"
	@echo "  go-vet       - Run go vet for all backend packages"
	@echo "  go-test      - Run go test for all backend packages"
	@echo "  format       - Run go-vet, go-fmt, and go-lint"

backend:
	cd backend && go run ./cmd/auth &
	cd backend && go run ./cmd/chat &

frontend:
	cd frontend && npm run dev

go-fmt:
	@echo "Running go fmt..."
	cd backend && goimports -w .

go-vet:
	@echo "Running go vet..."
	cd backend && go vet ./...

go-lint:
	@echo "Running go lint..."
	cd backend && golangci-lint run ./...

format:  go-vet go-fmt go-lint
