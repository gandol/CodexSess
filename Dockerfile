# Build stage for frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy frontend package files
COPY web/package*.json ./

# Install frontend dependencies
RUN npm install

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build:web

# Build stage for backend
FROM golang:1.26-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from frontend-builder
# Note: Vite builds directly to ../internal/webui/assets, so we copy from there
COPY --from=frontend-builder /app/internal/webui/assets ./internal/webui/assets

# Build backend binary
ARG APP_VERSION=docker
RUN go build -ldflags "-X main.appVersion=${APP_VERSION}" -o codexsess .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies including Node.js for codex CLI
RUN apk add --no-cache ca-certificates tzdata nodejs npm

# Create non-root user
RUN addgroup -g 1000 codexsess && \
    adduser -D -u 1000 -G codexsess codexsess

# Copy binary from builder
COPY --from=backend-builder /app/codexsess .

# Install codex CLI globally
RUN npm install -g @openai/codex

# Create data directory
RUN mkdir -p /app/data && chown -R codexsess:codexsess /app

# Switch to non-root user
USER codexsess

# Expose default port
EXPOSE 3061

# Set environment variables
ENV PORT=3061

# Run the application
CMD ["./codexsess"]
