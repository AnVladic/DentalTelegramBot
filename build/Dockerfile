FROM golang:1.23.1 AS builder
WORKDIR /app

COPY ../go.* ./
RUN go mod download

COPY ../ ./
RUN go build -v -o bot cmd/main.go

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bot /app/bot
CMD ["/app/bot"]
