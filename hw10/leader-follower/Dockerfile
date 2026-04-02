FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod from root (shared dependency)
COPY go.mod go.sum ./
RUN go mod download

# Copy shared module and leader-follower code
COPY shared/ ./shared/
COPY leader-follower/ ./leader-follower/

# Build the binary
RUN go build -o /server ./leader-follower/

# --- Runtime stage ---
FROM alpine:3.19
COPY --from=builder /server /server
EXPOSE 8080
CMD ["/server"]
