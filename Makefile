.PHONY: up down ingest api coordinator store tidy build

up:
	docker compose up -d

down:
	docker compose down

ingest:
	go run ./cmd/ingestion

api:
	go run ./cmd/api

# Run a local store node. Override NODE_ID/PORT/DATA_DIR as needed.
store:
	NODE_ID=store-1 PORT=7001 DATA_DIR=./data/store-1 go run ./cmd/store

coordinator:
	go run ./cmd/coordinator

coordinator-distributed:
	STORE_NODES=localhost:7001,localhost:7002,localhost:7003 go run ./cmd/coordinator

web:
	cd web && npm install && npm run dev

tidy:
	go mod tidy

build:
	go build -o bin/ingestion ./cmd/ingestion
	go build -o bin/api ./cmd/api
	go build -o bin/coordinator ./cmd/coordinator
	go build -o bin/store ./cmd/store

# Run 3 store nodes + api in distributed mode (requires separate terminals)
store-1:
	NODE_ID=store-1 PORT=7001 DATA_DIR=./data/store-1 go run ./cmd/store

store-2:
	NODE_ID=store-2 PORT=7002 DATA_DIR=./data/store-2 go run ./cmd/store

store-3:
	NODE_ID=store-3 PORT=7003 DATA_DIR=./data/store-3 go run ./cmd/store

api-distributed:
	STORE_NODES=localhost:7001,localhost:7002,localhost:7003 go run ./cmd/api
