COMPOSE := docker compose -f deployments/compose/compose.local.yaml

.PHONY: help structure verify up down ps logs reset

help:
	@echo "make structure  Tampilkan struktur repository"
	@echo "make verify     Verifikasi workspace"
	@echo "make up         Jalankan local stack"
	@echo "make down       Hentikan local stack"
	@echo "make ps         Tampilkan status local stack"
	@echo "make logs       Ikuti log local stack"
	@echo "make reset      Hapus local stack beserta datanya"

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
	@$(COMPOSE) config --quiet
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