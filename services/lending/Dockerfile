# Build stage
FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/lending ./cmd/api

# Runtime stage
FROM alpine:3.21

RUN adduser -D -u 10001 -s /sbin/nologin appuser
USER appuser

WORKDIR /app
COPY --from=builder /bin/lending /bin/lending

EXPOSE 8083

ENTRYPOINT ["/bin/lending"]
