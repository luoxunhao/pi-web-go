.PHONY: build build-frontend frontend-embed run dev test clean

FRONTEND_DIST := frontend/dist
EMBED_DIST := internal/webui/dist

# build embeds the built frontend into the binary (decision E1: production
# go:embed; when frontend/dist is missing the binary falls back to the
# configured disk frontend_dir, which is the development mode).
build: frontend-embed
	go build -o pi-web-go.exe ./cmd/server

build-frontend:
	cd frontend && npm run build

# Copies the built frontend into the go:embed tree. No-op when the frontend
# has not been built yet.
frontend-embed:
	@if [ -f $(FRONTEND_DIST)/index.html ]; then \
		mkdir -p $(EMBED_DIST) && cp -r $(FRONTEND_DIST)/. $(EMBED_DIST)/ && \
		echo "embedded frontend: $(FRONTEND_DIST) -> $(EMBED_DIST)"; \
	else \
		echo "warning: $(FRONTEND_DIST)/index.html missing; embed skipped (disk frontend_dir fallback)"; \
	fi

run:
	go run ./cmd/server

dev:
	cd frontend && npm run dev

test:
	go test ./internal/... ./cmd/...
	cd frontend && npm run typecheck

clean:
	rm -f pi-web-go.exe
	rm -rf frontend/dist
	@find $(EMBED_DIST) -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null || true
