# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.4

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cron ./cmd/cron

FROM gcr.io/distroless/static-debian12:nonroot AS runtime-base

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY config.yaml /app/config.yaml
COPY migrations /app/migrations
COPY locales /app/locales
COPY templates /app/templates

ENV CONFIG_PATH=/app/config.yaml

FROM runtime-base AS api

COPY --from=builder /out/api /app/app

EXPOSE 1300 9091

ENTRYPOINT ["/app/app"]

FROM runtime-base AS worker

COPY --from=builder /out/worker /app/app

EXPOSE 9091

ENTRYPOINT ["/app/app"]

FROM runtime-base AS cron

COPY --from=builder /out/cron /app/app

EXPOSE 9091

ENTRYPOINT ["/app/app"]
