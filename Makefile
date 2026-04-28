.PHONY: up down logs build tidy

# Start the full stack (Redis, 3 store nodes, ingestion, coordinator, API, dashboard)
up:
	docker compose up --build

# Start in background
up-detached:
	docker compose up --build -d

down:
	docker compose down

# Stream logs for a specific service: make logs s=api
logs:
	docker compose logs -f $(s)

build:
	export PATH=$$PATH:/usr/local/go/bin && \
	go build -o bin/ingestion ./cmd/ingestion && \
	go build -o bin/api ./cmd/api && \
	go build -o bin/coordinator ./cmd/coordinator && \
	go build -o bin/store ./cmd/store

tidy:
	export PATH=$$PATH:/usr/local/go/bin && go mod tidy
