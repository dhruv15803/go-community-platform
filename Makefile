
MIGRATIONS_DIR = ./internal/database/migrations
DATABASE_CONN=postgresql://postgres:1234%40%23A@localhost:5432/go-community-platform-db?sslmode=disable


.PHONY:all


start:
	go run ./cmd/api .

start_worker:
	go run ./cmd/worker .

create_admin:
	go run ./cmd/createAdminUser/main.go --email $(EMAIL) --password $(PASSWORD)


create_migration:
	migrate create -ext sql -dir $(MIGRATIONS_DIR)  -seq $(MIGRATION_NAME)


migrate_up:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_CONN) up

migrate_down:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_CONN) down



