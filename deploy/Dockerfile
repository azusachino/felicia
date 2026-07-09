# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically compiled binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/felicia-api ./cmd/api

# Stage 2: Runtime container
FROM alpine:latest

# Install CA certificates for HTTPS requests to Immich/Dawarich APIs
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the compiled binary from builder stage
COPY --from=builder /app/felicia-api /app/felicia-api

EXPOSE 8080

ENTRYPOINT ["/app/felicia-api"]
