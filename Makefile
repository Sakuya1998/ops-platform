.PHONY: all build dev proto docker-up docker-down clean

all: proto build

# ============ Proto ============
proto:
	@echo "Generating protobuf code..."
	@powershell -File scripts/gen-proto.ps1

# ============ Build ============
build: build-auth build-iam build-audit build-notify

build-auth:
	cd services/auth-svc && go build -o ../../bin/auth-svc ./cmd/main.go

build-iam:
	cd services/iam-svc && go build -o ../../bin/iam-svc ./cmd/main.go

build-audit:
	cd services/audit-svc && go build -o ../../bin/audit-svc ./cmd/main.go

build-notify:
	cd services/notify-svc && go build -o ../../bin/notify-svc ./cmd/main.go

# ============ Docker ============
docker-up:
	docker-compose -f deploy/docker-compose.yml up -d

docker-down:
	docker-compose -f deploy/docker-compose.yml down

docker-dev:
	docker-compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d

# ============ Dev ============
dev-auth:
	cd services/auth-svc && go run ./cmd/main.go

dev-iam:
	cd services/iam-svc && go run ./cmd/main.go

dev-audit:
	cd services/audit-svc && go run ./cmd/main.go

dev-notify:
	cd services/notify-svc && go run ./cmd/main.go

dev-web:
	cd web && npm run dev

# ============ Clean ============
clean:
	rm -rf bin/

# ============ Test ============
test:
	go test ./pkg/... ./services/...

test-auth:
	cd services/auth-svc && go test ./...

test-iam:
	cd services/iam-svc && go test ./...

test-audit:
	cd services/audit-svc && go test ./...

test-notify:
	cd services/notify-svc && go test ./...

test-web:
	cd web && npm test
