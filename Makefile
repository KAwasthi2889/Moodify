.PHONY: dev build test docker-up docker-down setup-python clean

# ── Go ──────────────────────────────────────────────

dev:
	go run ./cmd/server

# Start both Backend and Frontend concurrently with one command
dev-all:
	@(trap 'kill 0' SIGINT; go run ./cmd/server & (cd frontend && npm run dev))

build:
	go build -o bin/moodify ./cmd/server

test:
	go test ./... -v -race

test-e2e:
	go test ./tests/e2e/... -v


# ── Docker ──────────────────────────────────────────

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-reset: docker-down
	docker volume rm moodify_pgdata 2>/dev/null || true
	$(MAKE) docker-up

# ── Python ──────────────────────────────────────────

setup-python:
	python3 -m venv python/.venv
	python/.venv/bin/pip install --upgrade pip
	python/.venv/bin/pip install -r python/requirements.txt

# ── Utilities ───────────────────────────────────────

clean:
	rm -rf bin/ uploads/

env:
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; else echo ".env already exists"; fi
