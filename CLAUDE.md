# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ripple is a content automation distribution tool that processes structured notes from Notion and automatically distributes them to multiple platforms (social media, blogs, WeChat Official Account, etc.). The name reflects the idea that your thoughts create ripples of influence across platforms.

## Key Features

- **Notion Integration**: Syncs notes via Notion API
- **Content Processing**: Title optimization, summary generation, tag extraction, multi-platform template rendering
- **Multi-Platform Distribution**: Twitter/X, WeChat Official Account, XiaoHongShu, Blog platforms (Hugo, Ghost, Notion Blog), Email (Mailchimp/Substack)
- **AI Enhancement**: Optional content polishing, splitting into multiple posts, intelligent summarization

## Architecture

The system follows a pipeline architecture:
1. **Content Fetcher**: Retrieves notes from Notion
2. **Content Parser & Structurer**: Processes and structures content
3. **Multi-Platform Template Renderer**: Adapts content for different platforms
4. **Distribution Module**: Publishes to target platforms

## Tech Stack

- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **HTTP Framework**: Gin
- **Database**: PostgreSQL with GORM
- **Logging**: Zap with file rotation (lumberjack)
- **Configuration**: YAML with environment variable support (go-yaml-env)
- **External APIs**: Notion API for content fetching

## Development Commands

```bash
# Install dependencies
make install

# Build the application
make build

# Run the application
make run

# Development mode with auto-reload (requires air)
make dev

# Run tests
make test

# Clean build artifacts
make clean

# Tidy dependencies
make tidy
```

## Configuration

Configuration is managed through `configs/server.yaml` with environment variable support:

```yaml
server:
  host: "${HOST:localhost}"
  port: ${PORT:5334}
  mode: "${GIN_MODE:debug}"

database:
  host: "${DB_HOST:localhost}"
  port: ${DB_PORT:5432}
  username: "${DB_USERNAME:postgres}"
  password: "${DB_PASSWORD:postgres}"
  database: "${DB_DATABASE:ripple}"

notion:
  token: "${NOTION_TOKEN:}"
  database_id: "${NOTION_DATABASE_ID:}"
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/notion/pages` - Get all synced Notion pages
- `POST /api/v1/notion/sync` - Sync pages from Notion database

## Project Structure

```
├── cmd/server/          # Main application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── models/         # Database models
│   ├── server/         # HTTP server setup
│   └── service/        # Business logic services
├── pkg/logger/         # Shared logger package
├── configs/            # Configuration files
├── logs/              # Log files
└── bin/               # Compiled binaries
```

## Database Models

- `NotionPage`: Stores synced pages from Notion
- `DistributionJob`: Tracks distribution jobs to platforms
- `Platform`: Configuration for different distribution platforms

## Development Notes

- The service automatically migrates database schema on startup
- Notion API integration supports pagination and incremental sync
- All configuration supports environment variables with defaults
- Structured logging with multiple output formats (console/JSON)