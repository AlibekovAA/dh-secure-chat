ifeq ($(OS),Windows_NT)
	DOCKER_COMPOSE = docker-compose
else
	ifeq ($(shell command -v docker-compose 2> /dev/null),)
		DOCKER_COMPOSE = docker compose
	else
		DOCKER_COMPOSE = docker-compose
	endif
endif

COMPOSE_FILES = -f docker-compose.yml \
	-f yaml/db.yml \
	-f yaml/auth.yml \
	-f yaml/chat.yml \
	-f yaml/frontend.yml \
	-f yaml/proxy.yml

.PHONY: down down-volumes up up-build restart reup rebuild clean help backend frontend go-fmt go-vet go-test format

down:
	@echo "Stopping containers..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) down
	@echo "Done!"

down-volumes:
	@echo "Stopping containers and removing volumes..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) down -v
	@echo "Done!"

up:
	@echo "Starting containers..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) up

up-build:
	@echo "Starting containers with build..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) up --build

restart:
	@echo "Restarting containers (keeps volumes and cache)..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) restart
	@echo "Done!"

reup:
	@echo "Stopping containers and removing volumes..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) down -v
	@echo "Starting containers with build..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) up --build

rebuild:
	@echo "Rebuilding images with no cache..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) build --no-cache
	@echo "Starting containers..."
	$(DOCKER_COMPOSE) $(COMPOSE_FILES) up

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
	@echo "  up           - Start containers (fast, no build)"
	@echo "  up-build     - Start containers with build"
	@echo "  down         - Stop containers (keeps volumes)"
	@echo "  down-volumes - Stop containers and remove volumes (database will be removed!)"
	@echo "  restart      - Restart containers (fastest, keeps everything)"
	@echo "  reup         - Stop + rebuild + start (keeps volumes/database)"
	@echo "  rebuild      - Full rebuild without cache (slowest)"
	@echo "  clean        - Remove ALL Docker containers, images, and volumes"
	@echo "  help         - Show this help message"
	@echo "  backend      - Run backend services locally without Docker"
	@echo "  frontend     - Run frontend locally without Docker"
	@echo "  go-fmt       - Run go fmt for all backend packages"
	@echo "  go-vet       - Run go vet for all backend packages"
	@echo "  go-test      - Run go test for all backend packages"
	@echo "  format       - Alias for go-fmt"

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
