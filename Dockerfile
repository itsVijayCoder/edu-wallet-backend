# ── Stage 1: Build ────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/eduwallet-migrate ./cmd/migrate

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /bin/eduwallet-migrate ./eduwallet-migrate
COPY --from=builder /src/migrations ./migrations
COPY render-start.sh ./render-start.sh
RUN chmod +x render-start.sh

EXPOSE 8080

CMD ["./api"]
