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

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S eduwallet \
	&& adduser -S -G eduwallet -h /app eduwallet

WORKDIR /app

COPY --chown=eduwallet:eduwallet --from=builder /bin/api ./api
COPY --chown=eduwallet:eduwallet --from=builder /bin/eduwallet-migrate ./eduwallet-migrate
COPY --chown=eduwallet:eduwallet --from=builder /src/migrations ./migrations
COPY --chown=eduwallet:eduwallet render-start.sh ./render-start.sh
RUN chmod +x render-start.sh

EXPOSE 8080

USER eduwallet

CMD ["./render-start.sh"]
