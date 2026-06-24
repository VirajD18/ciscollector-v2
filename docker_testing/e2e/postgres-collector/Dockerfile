# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ciscollector ./cmd/ciscollector

FROM postgres:17

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl gettext-base && \
    rm -rf /var/lib/apt/lists/*

ENV POSTGRES_USER=shielduser
ENV POSTGRES_PASSWORD=shieldpass
ENV POSTGRES_DB=shielddb

WORKDIR /app
COPY --from=builder /out/ciscollector /app/ciscollector
COPY docker_testing/e2e/postgres-collector/entrypoint.sh /app/entrypoint.sh
COPY docker_testing/e2e/postgres-collector/kshieldconfig.toml.template /app/kshieldconfig.toml.template
COPY docker_testing/migrations/*.up.sql /docker-entrypoint-initdb.d/
COPY docker_testing/e2e/postgres-collector/init/99-bootstrap-users.sh /docker-entrypoint-initdb.d/99-bootstrap-users.sh
COPY docker_testing/e2e/postgres-collector/init/98-enable-logging.sh /docker-entrypoint-initdb.d/98-enable-logging.sh

RUN chmod +x /app/ciscollector /app/entrypoint.sh /docker-entrypoint-initdb.d/99-bootstrap-users.sh /docker-entrypoint-initdb.d/98-enable-logging.sh

EXPOSE 5432
ENTRYPOINT ["/app/entrypoint.sh"]
