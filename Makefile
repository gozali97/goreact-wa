# ─── WA Proxy Platform — Makefile ─────────────────────────
# Note: `make dev` runs backend + frontend dev servers. On Windows use Git Bash
# or WSL, or run the two commands manually (see README).

FRONTEND_DIR := apps/frontend
BACKEND_PKG  := ./apps/backend
EMBED_DIR    := internal/webui/dist
BIN          := bin/wa-proxy

.PHONY: help deps build build-frontend build-backend dev dev-backend dev-frontend run clean tidy vet

help:
	@echo "Targets:"
	@echo "  deps           Install Go modules and frontend npm packages"
	@echo "  build          Build frontend then the single Go binary (embedded)"
	@echo "  build-frontend Build the React app into $(EMBED_DIR)"
	@echo "  build-backend  Build the Go binary (expects frontend already built)"
	@echo "  dev            Run backend and frontend dev servers together"
	@echo "  run            Run the compiled binary"
	@echo "  vet            go vet ./..."
	@echo "  tidy           go mod tidy"
	@echo "  clean          Remove build artifacts"

deps:
	go mod download
	cd $(FRONTEND_DIR) && npm install

build-frontend:
	cd $(FRONTEND_DIR) && npm run build

build-backend:
	go build -o $(BIN) $(BACKEND_PKG)

build: build-frontend build-backend
	@echo "Built $(BIN) with embedded dashboard."

dev-backend:
	go run $(BACKEND_PKG)

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

# Run both dev servers concurrently (requires a POSIX shell).
dev:
	@echo "Starting backend (:8080) and frontend (:5173)..."
	@$(MAKE) -j2 dev-backend dev-frontend

run:
	$(BIN)

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin $(EMBED_DIR)/assets $(EMBED_DIR)/index.html
