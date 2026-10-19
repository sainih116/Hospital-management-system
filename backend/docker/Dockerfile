# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/hms-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/hms-seed ./cmd/seed

# ---- Runtime stage ----
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S hms && adduser -S hms -G hms

WORKDIR /app

COPY --from=builder /out/hms-api ./hms-api
COPY --from=builder /out/hms-seed ./hms-seed
COPY --from=builder /src/docs ./docs

ENV PORT=5000
EXPOSE 5000

USER hms

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:5000/health || exit 1

ENTRYPOINT ["./hms-api"]
