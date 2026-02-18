.PHONY: help run build db-setup db-migrate db-rollback db-drop clean test

# Default target
help:
	@echo "Available commands:"
	@echo "  make run         - Run the bot"
	@echo "  make build       - Build the bot binary"
	@echo "  make db-setup    - Create database and run migrations"
	@echo "  make db-migrate  - Run database migrations"
	@echo "  make db-rollback - Rollback last migration"
	@echo "  make db-drop     - Drop database"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make test        - Run tests"

# Run the bot
run:
	go run cmd/main.go

# Build the bot
build:
	go build -o bot cmd/bot/main.go

# Create database and run migrations
db-setup:
	@echo "Creating database if it doesn't exist..."
	@createdb telegram_bot 2>/dev/null || true
	@echo "Running migrations..."
	@psql -d telegram_bot -f migrations/001_initial_schema.up.sql
	@echo "Database setup complete!"

# Run migrations (automated via golang-migrate)
db-migrate:
	@echo "Migrations run automatically when bot starts"
	@echo "Or you can run manually: psql -d telegram_bot -f migrations/001_initial_schema.up.sql"

# Rollback migrations
db-rollback:
	@psql -d telegram_bot -f migrations/001_initial_schema.down.sql
	@echo "Migration rolled back"

# Drop database
db-drop:
	@dropdb telegram_bot 2>/dev/null || true
	@echo "Database dropped"

# Clean build artifacts
clean:
	@rm -f bot
	@echo "Build artifacts cleaned"

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
docker-up:
	docker compose up -d --no-deps --build bot