.PHONY: up down ingest api coordinator tidy build

up:
	docker compose up -d

down:
	docker compose down

ingest:
	go run ./cmd/ingestion

api:
	go run ./cmd/api

coordinator:
	go run ./cmd/coordinator

tidy:
	go mod tidy

build:
	go build -o bin/ingestion ./cmd/ingestion
	go build -o bin/api ./cmd/api
	go build -o bin/coordinator ./cmd/coordinator
