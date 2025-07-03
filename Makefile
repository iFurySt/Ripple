.PHONY: build run dev test clean install tidy

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
	@docker run -p 5334:5334 ripple:latest