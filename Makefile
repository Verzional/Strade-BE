# ==========================================
# POLYGLOT MONOREPO ORCHESTRATOR
# ==========================================

.PHONY: help dev down build restart clean test-all shell-core shell-schedule shell-chat

# Default action when just typing 'make'
help:
	@echo "Available commands:"
	@echo "  make dev            - Start all databases and microservices locally (with hot-reload)"
	@echo "  make down           - Stop all running containers and preserve data"
	@echo "  make build          - Force a complete rebuild of all service Docker images"
	@echo "  make restart        - Quick restart of all services"
	@echo "  make clean          - Stop containers and completely wipe out database volumes"
	@echo "  make test-all       - Run test suites across TS, Python, and Go services"
	@echo "  make shell-core     - Drop into the Core Postgres Interactive Shell"
	@echo "  make shell-schedule - Drop into the Schedule Postgres Interactive Shell"
	@echo "  make shell-chat     - Drop into the MongoDB Interactive Shell"

# ------------------------------------------
# DOCKER ORCHESTRATION
# ------------------------------------------

dev:
	docker compose up --build

down:
	docker compose down

build:
	docker compose build --no-cache

restart:
	docker compose restart

clean:
	docker compose down -v
	@echo "All containers dropped and volumes wiped clean."

# ------------------------------------------
# DATABASE INTERACTION SHORTCUTS
# ------------------------------------------

shell-core:
	docker exec -it db-postgres-core psql -U admin -d core_db

shell-schedule:
	docker exec -it db-postgres-schedule psql -U admin -d schedule_db

shell-chat:
	docker exec -it db-mongodb-chat mongosh -u admin -p password --authenticationDatabase admin

# ------------------------------------------
# MULTI-LANGUAGE QUALITY ASSURANCE
# ------------------------------------------

test-all:
	@echo "=== Running TypeScript Tests (User, Skill) ==="
	pnpm --filter user test || true
	pnpm --filter skill test || true
	@echo "=== Running Python Tests (Review, Schedule) ==="
	cd services/review && uv run pytest || true
	cd services/schedule && uv run pytest || true
	@echo "=== Running Go Tests (Chat, Gateway) ==="
	cd services/chat && go test ./... || true
	cd services/gateway && go test ./... || true