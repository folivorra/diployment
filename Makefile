include config/.pg.env

export GOOSE_DRIVER := postgres
export GOOSE_DBSTRING := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
export GOOSE_MIGRATION_DIR := ./migrations

.PHONY: cluster_up cluster_down migrate_up

all: cluster_up

cluster_up:
	docker compose up --build -d

cluster_down:
	docker compose down

migrate_up:
	goose up

migrate_down:
	goose down