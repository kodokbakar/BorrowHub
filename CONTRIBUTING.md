# Contributing to BorrowHub

Thank you for contributing to BorrowHub.

BorrowHub is intended to be understandable, operationally realistic, and friendly to new contributors. Changes should solve a concrete product, reliability, security, or contributor problem without adding unnecessary complexity.

## Before starting

For significant changes:

1. Search existing issues and pull requests.
2. Open an issue describing the problem.
3. Discuss changes to service boundaries, contracts, database schemas, security, or deployment behavior before implementation.

Small documentation fixes and isolated bug fixes may be submitted directly.

## Local setup

Install the required tools listed in `README.md`, then run:

```bash
make bootstrap
make up
make seed
make verify
```

Run the applications with:

```bash
make dev-web
make dev-identity
make dev-inventory
make dev-lending
```

## Branches

Create a short-lived branch from `main`.

Examples:

```text
feat/inventory-categories
fix/request-id-header
docs/local-setup
test/reservation-conflict
```

Do not work directly on `main`.

## Code expectations

* Prefer the standard library and existing project dependencies.
* Avoid abstractions with only one speculative use case.
* Keep domain logic independent from HTTP handlers.
* Keep database access inside the owning service.
* Do not query another service's database.
* Validate input at trust boundaries.
* Enforce authorization in backend services.
* Never log passwords, tokens, cookies, or sensitive personal data.
* Use transactions for multi-step invariants.
* Use database constraints for concurrency-sensitive guarantees.
* Keep network calls bounded by timeouts.
* Make retryable commands idempotent.

## Contracts

Update OpenAPI contracts before or together with HTTP behavior changes.

Update event contracts before or together with event changes.

Breaking changes require explicit review and a documented migration path.

Validate contracts with:

```bash
make contracts
```

## Database changes

Database migrations belong to the service that owns the data.

A migration must:

* Apply cleanly to an empty database.
* Preserve tenant isolation.
* Avoid destructive changes without an expand-and-contract plan.
* Include rollback considerations.
* Be tested before production use.

## Testing

Run:

```bash
make verify
```

Business-critical changes should include tests for the behavior being changed.

Important areas include:

* Authorization
* Tenant isolation
* Reservation conflicts
* Idempotency
* Status transitions
* Retry behavior
* Event deduplication
* Public field redaction

## Commits

Use clear, imperative commit messages.

Examples:

```text
feat: add category creation endpoint
fix: reject cross-organization inventory access
test: cover concurrent reservation conflict
docs: clarify local database reset
ci: validate event contracts
```

Keep commits focused and reviewable.

## Pull requests

A pull request should explain:

* The problem.
* The chosen solution.
* The affected services.
* Contract or event changes.
* Database migration impact.
* Security impact.
* Tests performed.
* Rollback considerations.
* Known limitations.

Frontend changes should include screenshots when useful.

Do not merge security-sensitive changes or major migrations without another reviewer.

## Dependencies

A new dependency must solve a current problem that cannot be handled reasonably by:

1. Existing code.
2. The standard library.
3. Native browser or database features.
4. An already-installed dependency.

Do not add dependencies for speculative future use.

## Security

Do not report vulnerabilities in public issues.

Follow `SECURITY.md` for private reporting.