# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server

# Final stage - use minimal image
FROM alpine:latest

# Install CA certificates for HTTPS connections to AWS
RUN apk --no-cache add ca-certificates

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]
