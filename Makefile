COMPOSE := docker compose -f deployments/compose/compose.local.yaml

.PHONY: \
	help \
	structure \
	verify \
	up \
	down \
	ps \
	logs \
	reset \
	dev-web \
	dev-identity \
	dev-inventory \
	dev-lending \
	health

help:
	@echo "make structure       Tampilkan struktur repository"
	@echo "make verify          Verifikasi workspace"
	@echo "make up              Jalankan local stack"
	@echo "make down            Hentikan local stack"
	@echo "make ps              Tampilkan status local stack"
	@echo "make logs            Ikuti log local stack"
	@echo "make reset           Hapus local stack beserta datanya"
	@echo "make dev-web         Jalankan frontend"
	@echo "make dev-identity    Jalankan Identity Service"
	@echo "make dev-inventory   Jalankan Inventory Service"
	@echo "make dev-lending     Jalankan Lending Service"
	@echo "make health          Periksa health seluruh service"

structure:
	@find . \
		-maxdepth 4 \
		-not -path './.git*' \
		-not -path './node_modules*' \
		| sort

verify:
	@test -f package.json
	@test -f bun.lock
	@test -f go.work
	@test -f services/identity/go.mod
	@test -f services/inventory/go.mod
	@test -f services/lending/go.mod
	@test -f workers/notification/go.mod
	@go work edit -json >/dev/null
	@go test ./services/identity/...
	@go test ./services/inventory/...
	@go test ./services/lending/...
	@$(COMPOSE) config --quiet
	@$(COMPOSE) run --rm --no-deps caddy \
		caddy validate \
		--config /etc/caddy/Caddyfile
	@echo "Workspace verification passed"

up:
	@$(COMPOSE) up -d --wait

down:
	@$(COMPOSE) down

ps:
	@$(COMPOSE) ps

logs:
	@$(COMPOSE) logs -f

reset:
	@$(COMPOSE) down -v

dev-web:
	@bun run dev:web

dev-identity:
	@APP_ENV=local HTTP_ADDR=:8081 \
		go run ./services/identity/cmd/api

dev-inventory:
	@APP_ENV=local HTTP_ADDR=:8082 \
		go run ./services/inventory/cmd/api

dev-lending:
	@APP_ENV=local HTTP_ADDR=:8083 \
		go run ./services/lending/cmd/api

health:
	@curl --fail --silent --show-error \
		http://identity.borrowhub.localhost:8080/health/live
	@echo
	@curl --fail --silent --show-error \
		http://identity.borrowhub.localhost:8080/health/ready
	@echo
	@curl --fail --silent --show-error \
		http://inventory.borrowhub.localhost:8080/health/live
	@echo
	@curl --fail --silent --show-error \
		http://inventory.borrowhub.localhost:8080/health/ready
	@echo
	@curl --fail --silent --show-error \
		http://lending.borrowhub.localhost:8080/health/live
	@echo
	@curl --fail --silent --show-error \
		http://lending.borrowhub.localhost:8080/health/ready
	@echo

.PHONY: test test-integration

test:
	@go test -timeout=30s ./services/identity/...
	@go test -timeout=30s ./services/inventory/...
	@go test -timeout=30s ./services/lending/...
	@go test -timeout=30s ./workers/notification/...

test-integration: up
	@DATABASE_URL='postgres://borrowhub_identity:borrowhub_identity@127.0.0.1:5432/borrowhub_identity?sslmode=disable' \
		EXPECTED_DATABASE='borrowhub_identity' \
		go test -count=1 -tags=integration -timeout=30s ./services/identity/...
	@DATABASE_URL='postgres://borrowhub_inventory:borrowhub_inventory@127.0.0.1:5432/borrowhub_inventory?sslmode=disable' \
		EXPECTED_DATABASE='borrowhub_inventory' \
		go test -count=1 -tags=integration -timeout=30s ./services/inventory/...
	@DATABASE_URL='postgres://borrowhub_lending:borrowhub_lending@127.0.0.1:5432/borrowhub_lending?sslmode=disable' \
		EXPECTED_DATABASE='borrowhub_lending' \
		go test -count=1 -tags=integration -timeout=30s ./services/lending/...
	@DATABASE_URL='postgres://borrowhub_notification:borrowhub_notification@127.0.0.1:5432/borrowhub_notification?sslmode=disable' \
		EXPECTED_DATABASE='borrowhub_notification' \
		go test -count=1 -tags=integration -timeout=30s ./workers/notification/...