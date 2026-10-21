.PHONY: all build build-frontend build-backend test run docker-build docker-up clean

all: build

build: build-frontend build-backend

build-frontend:
	cd frontend && npm install && npm run build

build-backend:
	cd backend && go build -o rss2kindle ./cmd/rss2kindle

test:
	cd backend && go test -v ./...

run: build
	R2K_ADMIN_PASSWORD=admin ./backend/rss2kindle

docker-build:
	docker build -t rss2kindle:latest .

docker-up:
	docker compose up -d

clean:
	rm -rf backend/rss2kindle backend/web/dist
