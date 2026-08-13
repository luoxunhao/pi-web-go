.PHONY: build build-frontend run dev test clean

build:
	go build -o pi-web-go.exe ./cmd/server

build-frontend:
	cd frontend && npm run build

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
