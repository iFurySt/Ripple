.PHONY: build run dev test clean install tidy web-install web-dev web-build web-clean dev-with-web build-all clean-all docker-build docker-run docker-run-with-env docker-stop docker-test docker-compose-up docker-compose-down

# Build the application
build:
	@echo "Building Ripple..."
	@go build -o bin/ripple cmd/server/main.go

# Run the application
run: build
	@echo "Running Ripple..."
	@./bin/ripple

# Development mode with auto-reload (requires air)
dev:
	@echo "Running in development mode..."
	@air -c .air.toml

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf logs/*.log

# Install dependencies
install:
	@echo "Installing dependencies..."
	@go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Database migration
migrate:
	@echo "Running database migration..."
	@./bin/ripple migrate

# Build for production
build-prod:
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/ripple-linux cmd/server/main.go

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t ripple:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 5334:5334 --env-file .env ripple:latest

# Docker run with environment
docker-run-with-env:
	@echo "Running Docker container with environment..."
	@docker run -p 5334:5334 --name ripple-test \
		--env-file .env \
		-v $(PWD)/logs:/opt/ripple/logs \
		ripple:latest

# Docker stop and remove test container
docker-stop:
	@echo "Stopping and removing test container..."
	@docker stop ripple-test || true
	@docker rm ripple-test || true

# Docker test - full cycle
docker-test: docker-build docker-stop docker-run-with-env

# Docker compose commands
docker-compose-up:
	@echo "Starting with docker-compose..."
	@docker-compose up --build

docker-compose-down:
	@echo "Stopping docker-compose..."
	@docker-compose down

# Web Dashboard commands
web-install:
	@echo "Installing web dependencies..."
	@cd web && npm install

web-dev:
	@echo "Starting web development server..."
	@cd web && npm run dev

web-build:
	@echo "Building web dashboard..."
	@cd web && npm run build

web-clean:
	@echo "Cleaning web artifacts..."
	@cd web && rm -rf dist node_modules

# Combined development (requires two terminals)
dev-with-web: build
	@echo "Starting backend server..."
	@echo "Run 'make web-dev' in another terminal for frontend development"
	@./bin/ripple

# Full build including web dashboard
build-all: build web-build
	@echo "Build completed with web dashboard"

# Clean everything including web
clean-all: clean web-clean
	@echo "All artifacts cleaned"