# Build stage for web frontend
FROM node:20.18.0 AS web-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Build stage for Go application
FROM golang:1.24.1 AS go-builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/ripple ./cmd/server

FROM ubuntu:22.04

WORKDIR /opt/ripple

# Copy the binary
COPY --from=go-builder /app/bin/ripple /opt/ripple/bin/ripple

# Copy web assets
COPY --from=web-builder /app/web/dist /opt/ripple/web/dist

# Copy configuration
COPY configs/ /opt/ripple/configs/

# Create logs directory
RUN mkdir -p /opt/ripple/logs

EXPOSE 5334

CMD ["/opt/ripple/bin/ripple"]
