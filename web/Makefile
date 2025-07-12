# Web Dashboard Makefile for Ripple

.PHONY: install dev build clean lint type-check preview

# Install dependencies
install:
	npm install

# Start development server
dev:
	npm run dev

# Build for production
build:
	npm run build

# Clean build artifacts
clean:
	rm -rf dist node_modules

# Run linting
lint:
	npm run lint

# Preview production build
preview:
	npm run preview

# Install dependencies and start development
dev-setup: install dev