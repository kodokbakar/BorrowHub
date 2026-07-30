# BorrowHub

BorrowHub is a multi-tenant equipment lending platform for small organizations.

It replaces spreadsheets, chat messages, and paper records with a structured workflow for inventory, loan requests, approvals, check-out, returns, overdue tracking, and notifications.

## Project status

BorrowHub is currently in the foundation milestone.

The repository contains:

* A React web application.
* Identity, Inventory, and Lending services.
* A Notification Worker.
* PostgreSQL databases owned by each service.
* RabbitMQ for asynchronous domain events.
* Caddy for local routing.
* OpenAPI and event contracts.
* CI and basic security checks.

## Architecture

BorrowHub uses a small microservice architecture:

| Component           | Responsibility                                                          |
| ------------------- | ----------------------------------------------------------------------- |
| Identity Service    | Users, sessions, organizations, memberships, and invitations            |
| Inventory Service   | Categories, items, physical units, condition, and maintenance           |
| Lending Service     | Loan requests, reservations, check-out, returns, and overdue state      |
| Notification Worker | In-app notifications, email delivery, retries, and dead-letter handling |
| Web                 | Public, member, and administrator interfaces                            |

Each service owns its own PostgreSQL database. Services must not query another service's database.

Synchronous operations use HTTP. Asynchronous domain events use RabbitMQ.

## Requirements

Install:

* Git
* Bun
* Go
* Docker Engine
* Docker Compose
* GNU Make
* curl

Optional development tools:

* `bat`
* `wl-copy`
* GitHub CLI

## Local setup

Clone the repository and enter the project directory:

```bash
git clone <repository-url>
cd borrowhub
```

Install dependencies:

```bash
make bootstrap
```

Start PostgreSQL, RabbitMQ, Mailpit, and Caddy:

```bash
make up
```

Create minimal development seed records:

```bash
make seed
```

Run each backend service in a separate terminal:

```bash
make dev-identity
```

```bash
make dev-inventory
```

```bash
make dev-lending
```

Run the frontend in another terminal:

```bash
make dev-web
```

Verify service health:

```bash
make health
```

## Local endpoints

| Component           | URL                                       |
| ------------------- | ----------------------------------------- |
| Web                 | http://borrowhub.localhost:8080           |
| Identity Service    | http://identity.borrowhub.localhost:8080  |
| Inventory Service   | http://inventory.borrowhub.localhost:8080 |
| Lending Service     | http://lending.borrowhub.localhost:8080   |
| RabbitMQ Management | http://rabbitmq.borrowhub.localhost:8080  |
| Mailpit             | http://mail.borrowhub.localhost:8080      |

RabbitMQ local credentials:

```text
Username: borrowhub
Password: borrowhub
```

Local credentials are development-only and must never be reused in staging or production.

## Common commands

```bash
make help
make bootstrap
make fmt
make test
make build
make contracts
make verify
make up
make down
make ps
make logs
make reset
make seed
make health
```

`make reset` deletes local PostgreSQL and RabbitMQ data.

## Repository structure

```text
borrowhub/
├── apps/
│   └── web/
├── services/
│   ├── identity/
│   ├── inventory/
│   └── lending/
├── workers/
│   └── notification/
├── contracts/
│   ├── openapi/
│   └── events/
├── deployments/
│   ├── compose/
│   └── caddy/
├── observability/
├── scripts/
├── docs/
│   └── adr/
└── .github/
    └── workflows/
```

## Contracts

HTTP contracts live in:

```text
contracts/openapi/
```

Domain event contracts live in:

```text
contracts/events/
```

Update the contract before or together with any implementation that changes API or event behavior.

Validate contracts with:

```bash
make contracts
```

## Development rules

* Keep service boundaries explicit.
* Do not share business tables across services.
* Enforce organization authorization in backend services.
* Prefer database constraints for data invariants.
* Keep domain events idempotent and free from unnecessary personal data.
* Do not add dependencies without a concrete use case.
* Add tests for business rules, security boundaries, and failure behavior.
* Run `make verify` before opening a pull request.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a change.

By participating in this project, contributors agree to follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

Report security issues using the process in [SECURITY.md](SECURITY.md).
