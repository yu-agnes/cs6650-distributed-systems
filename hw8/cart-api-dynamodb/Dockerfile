# ==================== Build stage ====================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cart-api .

# ==================== Run stage ====================
FROM alpine:3.21

WORKDIR /app
COPY --from=builder /app/cart-api .

EXPOSE 8080

CMD ["./cart-api"]
