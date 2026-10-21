# Build stage
FROM oven/bun:1.3.14-alpine AS builder
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

ENV PATH="/app/node_modules/.bin:$PATH"
COPY . .
RUN ln -s $(which bun) /usr/local/bin/node && bun run build

# Runtime stage
FROM caddy:2.11.3-alpine

RUN adduser -D -u 10001 -s /sbin/nologin appuser

WORKDIR /srv
COPY --from=builder /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile

USER appuser

EXPOSE 8080

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
