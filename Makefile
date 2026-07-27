.PHONY: help structure verify

help:
	@printf '%s\n' \
		'make structure  Tampilkan struktur repository' \
		'make verify     Verifikasi workspace awal'

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
	@printf '%s\n' 'M0-001 verification passed'