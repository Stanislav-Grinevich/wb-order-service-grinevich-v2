# собираем бинарники
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o wb-order-service .
RUN go build -o migrate ./cmd/migrate
RUN go build -o producer ./cmd/producer


# рантайм
FROM alpine:3.20

WORKDIR /app

# bash + netcat + pg_isready (postgres-client)
RUN apk add --no-cache ca-certificates bash netcat-openbsd postgresql-client

COPY --from=builder /app/wb-order-service /app/wb-order-service
COPY --from=builder /app/migrate /app/migrate
COPY --from=builder /app/producer /app/producer

COPY migrations /app/migrations
COPY web /app/web

COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 8081

ENTRYPOINT ["/app/entrypoint.sh"]
