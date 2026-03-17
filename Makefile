.PHONY: build build-frontend build-backend run dev kill clean test deps

BINARY_NAME ?= codexsess
BINARY_PATH ?= ./$(BINARY_NAME)
WEB_DIR ?= web
EMBED_ASSETS_DIR ?= internal/webui/assets
DEV_FRONTEND_PORT ?= 3051
DEV_BACKEND_PORT ?= 3052
RUN_PORT ?= 3061

build: build-frontend build-backend

build-frontend:
	@echo "Building frontend from $(WEB_DIR)..."
	cd $(WEB_DIR) && npm install && npm run build
	@echo "Frontend build output is configured directly to $(EMBED_ASSETS_DIR) via Vite outDir."

build-backend:
	@echo "Building backend binary $(BINARY_PATH)..."
	go build -o $(BINARY_PATH) .

run: build
	@echo "Starting $(BINARY_NAME) on port $(RUN_PORT)..."
	PORT=$(RUN_PORT) $(BINARY_PATH)

dev:
	@echo "Starting dev mode (frontend: $(DEV_FRONTEND_PORT), backend: $(DEV_BACKEND_PORT))..."
	@command -v air >/dev/null 2>&1 || { \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	}
	@set -e; \
	trap 'kill 0' INT TERM EXIT; \
	(cd $(WEB_DIR) && npm install && npm run dev -- --host 127.0.0.1 --port $(DEV_FRONTEND_PORT)) & \
	PORT=$(DEV_BACKEND_PORT) CODEXSESS_NO_OPEN_BROWSER=1 "$$(go env GOPATH)/bin/air" -c .air.toml

kill:
	@echo "Stopping codexsess dev/run processes on ports $(DEV_FRONTEND_PORT), $(DEV_BACKEND_PORT), $(RUN_PORT)..."
	-pkill -f "air -c .air.toml"
	-pkill -f "$(BINARY_NAME)"
	-pkill -f "vite.*$(DEV_FRONTEND_PORT)"
	-fuser -k $(DEV_FRONTEND_PORT)/tcp $(DEV_BACKEND_PORT)/tcp $(RUN_PORT)/tcp 2>/dev/null || true

clean:
	@echo "Cleaning build artifacts..."
	go clean
	rm -f $(BINARY_PATH)
	@echo "Directory cleanup skipped (no rm dir policy)."

test:
	go test ./...

deps:
	go mod download
	cd $(WEB_DIR) && npm install
