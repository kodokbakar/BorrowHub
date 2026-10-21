COMPOSE := docker compose -f deployments/compose/compose.local.yaml

GO_PACKAGES := \
	./services/identity/... \
	./services/inventory/... \
	./services/lending/... \
	./workers/notification/...

GO_BINARIES := \
	./services/identity/cmd/api \
	./services/inventory/cmd/api \
	./services/lending/cmd/api \
	./workers/notification/cmd/worker

.PHONY: \
	help \
	bootstrap \
	structure \
	fmt \
	fmt-check \
	vet \
	test \
	build \
	contracts \
	compose-check \
	verify \
	up \
	down \
	ps \
	logs \
	reset \
	seed \
	dev-web \
	dev-identity \
	dev-inventory \
	dev-lending \
	health

help:
	@echo "make bootstrap       Install dependency"
	@echo "make fmt             Format kode Go"
	@echo "make test            Jalankan backend test"
	@echo "make build           Build backend dan frontend"
	@echo "make contracts       Validasi API dan event contract"
	@echo "make verify          Jalankan seluruh pemeriksaan"
	@echo "make up              Jalankan local infrastructure"
	@echo "make down            Hentikan local infrastructure"
	@echo "make ps              Tampilkan status container"
	@echo "make logs            Ikuti log container"
	@echo "make reset           Hapus container dan local data"
	@echo "make seed            Tambahkan development seed"
	@echo "make dev-web         Jalankan frontend"
	@echo "make dev-identity    Jalankan Identity Service"
	@echo "make dev-inventory   Jalankan Inventory Service"
	@echo "make dev-lending     Jalankan Lending Service"
	@echo "make health          Periksa health semua service"
	@echo "make structure       Tampilkan struktur repository"

bootstrap:
	@bun install --frozen-lockfile

structure:
	@find . \
		-maxdepth 4 \
		-not -path './.git*' \
		-not -path './node_modules*' \
		| sort

fmt:
	@find services workers \
		-type f \
		-name '*.go' \
		-exec gofmt -w {} +

fmt-check:
	@test -z "$$(find services workers \
		-type f \
		-name '*.go' \
		-exec gofmt -l {} +)"

vet:
	@go vet $(GO_PACKAGES)

test:
	@go test -race $(GO_PACKAGES)

build:
	@go build $(GO_BINARIES)
	@bun run typecheck:web
	@bun run build:web

contracts:
	@docker run --rm \
		-v "$(CURDIR)/contracts/openapi:/spec" \
		redocly/cli:2.41.1 \
		lint \
		--config=/spec/redocly.yaml \
		/spec/identity.yaml \
		/spec/inventory.yaml \
		/spec/lending.yaml
	@for schema in contracts/events/*.json; do \
		python3 -m json.tool "$$schema" >/dev/null || exit 1; \
	done

compose-check:
	@$(COMPOSE) config --quiet
	@$(COMPOSE) run --rm --no-deps caddy \
		caddy validate \
		--config /etc/caddy/Caddyfile

verify: fmt-check vet test build contracts compose-check
	@go work edit -json >/dev/null
	@echo "Verification passed"

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

seed:
	@$(COMPOSE) up -d --wait postgres
	@$(COMPOSE) exec -T postgres \
		psql \
		-v ON_ERROR_STOP=1 \
		-U borrowhub \
		-d postgres \
		< scripts/seed/development.sql

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
