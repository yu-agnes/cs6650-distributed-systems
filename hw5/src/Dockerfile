# Build stage - specify amd64 architecture for AWS ECS
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files first (for better caching)
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary for linux/amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /product-api

# Runtime stage - specify amd64 architecture
FROM --platform=linux/amd64 alpine:latest

# Add ca-certificates for HTTPS (if needed)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /product-api .

# Expose port 8080
EXPOSE 8080

# Run the binary
CMD ["./product-api"]