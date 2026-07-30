# ADR 0001: BorrowHub Foundation Architecture

* Status: Accepted
* Date: 2026-07-30

## Context

BorrowHub needs to support real equipment lending workflows while remaining small enough for contributors to understand, run locally, test, and operate.

The foundation must establish clear service ownership, local development, contracts, deployment boundaries, and reliability expectations without introducing infrastructure that is not yet needed.

## Decision

BorrowHub will use a monorepo containing:

* One React web application.
* Identity, Inventory, and Lending backend services.
* One Notification Worker.
* Shared HTTP and event contracts.
* Local deployment and observability configuration.
* Contributor and operational documentation.

### Frontend

The web application uses:

* React.
* TypeScript.
* Vite.
* Bun.
* Tailwind CSS.

Remote server state will be managed by TanStack Query when API-backed features are implemented.

### Backend

Backend services use Go.

The foundation starts with the Go standard library HTTP server. A small router such as `chi` may be introduced when route composition or middleware requirements justify the dependency.

Each service owns its domain logic, HTTP boundary, persistence, configuration, and observability.

### Service boundaries

Identity Service owns:

* Users.
* Credentials.
* Sessions.
* Organizations.
* Memberships.
* Invitations.
* Roles.

Inventory Service owns:

* Categories.
* Equipment items.
* Physical units.
* Operational condition.
* Maintenance records.

Lending Service owns:

* Loan requests.
* Unit reservations.
* Loan lifecycle.
* Check-out tokens.
* Returns.
* Status history.
* Idempotency records.

Notification Worker owns:

* In-app notification records.
* Delivery attempts.
* Processed event records.
* Dead-letter handling.

### Data ownership

Each service uses its own PostgreSQL database and credentials.

Services must not query another service's database.

Cross-service identifiers are validated through contracts and service APIs rather than foreign keys across databases.

### Communication

HTTP is used when the caller requires an immediate answer.

RabbitMQ domain events are used for work that may happen asynchronously, including notifications and non-critical projections.

Events describe facts that already happened. Consumers must tolerate duplicate delivery and process events idempotently.

### Local development

Docker Compose runs:

* PostgreSQL.
* RabbitMQ.
* Mailpit.
* Caddy.

Application processes run directly on the development host during the foundation phase.

Caddy provides predictable `.localhost` routing.

### Contracts

Each backend service owns an OpenAPI contract.

Domain events use a shared JSON Schema envelope.

Contract changes are reviewed together with implementation changes.

### Deployment

The initial public deployment target is one VPS using Docker Compose and Caddy.

Kubernetes is deferred until measured operational requirements justify it.

### Reliability

The foundation includes:

* Liveness and readiness endpoints.
* Graceful shutdown.
* Request and correlation IDs.
* Structured JSON logging.
* Standard success and error envelopes.
* CI checks.
* Contract validation.
* Basic dependency and security scanning.

Database-backed readiness checks will be added when services open database pools.

### Security

Authorization is enforced by backend services.

Frontend route protection is not considered a security boundary.

Sensitive credentials, tokens, cookies, and personal data must not be written to logs or domain events.

## Consequences

### Positive

* Contributors can run the stack locally with a small number of commands.
* Service and database ownership are explicit.
* API and event changes are reviewable.
* Infrastructure remains understandable.
* Services can evolve independently where required.
* Notification failures do not need to block lending operations.

### Negative

* Multiple services increase local and operational complexity.
* Cross-service data cannot rely on database foreign keys.
* Event delivery requires retry and deduplication.
* A single VPS remains a production availability limitation.
* Some implementation is temporarily duplicated between services.

## Deliberate simplifications

* No Kubernetes.
* No service mesh.
* No shared business database.
* No distributed tracing during foundation work.
* No transactional outbox until domain event publishing exists.
* No shared Go framework before repeated code proves a stable abstraction.
* No domain seed data before domain schemas exist.

## Revisit triggers

Revisit this decision when:

* More than one production host is required.
* Independent horizontal scaling is measured as necessary.
* Availability targets cannot be met on one host.
* Repeated service code forms a stable shared boundary.
* Message publishing requires transactional outbox guarantees.
* Production debugging requires distributed tracing.
* The standard library router no longer keeps route composition clear.