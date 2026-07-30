# Security Policy

## Supported versions

BorrowHub is under active development.

Security fixes are applied to the current `main` branch and the latest supported release when releases become available.

Older development snapshots are not supported.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability.

Use GitHub private vulnerability reporting for the repository.

Include:

* A clear description of the vulnerability.
* The affected service, endpoint, or component.
* Reproduction steps.
* Expected and observed behavior.
* Potential impact.
* Any known workaround.
* Relevant logs with secrets and personal data removed.

Do not include:

* Passwords.
* Access tokens.
* Refresh tokens.
* Session cookies.
* Invitation tokens.
* Private user data.
* Production credentials.

## Safe testing

Security research must not:

* Access data belonging to other users or organizations.
* Modify or delete data without authorization.
* Disrupt service availability.
* Send spam or excessive traffic.
* Exfiltrate credentials or personal data.
* Test against production without explicit authorization.

Use local or explicitly approved test environments.

## Disclosure

Allow maintainers reasonable time to investigate and prepare a fix before public disclosure.

Maintainers may request additional information, reproduction assistance, or coordinated disclosure timing.

## Security-sensitive areas

Changes in these areas require additional review:

* Authentication and sessions.
* Organization membership and roles.
* Tenant isolation.
* Public catalog redaction.
* Password and token handling.
* CSRF, CORS, and cookie policy.
* Database migrations.
* RabbitMQ event contents.
* Logging and observability.
* Deployment credentials.
* Backup and restore procedures.