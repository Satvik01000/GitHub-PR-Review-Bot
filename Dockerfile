# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates and tzdata for static build requirements
RUN apk add --no-cache ca-certificates tzdata

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build lightweight, statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Final Stage
FROM alpine:3.21

WORKDIR /app

# Install CA certificates for HTTPS outbound requests to GitHub & AI APIs
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled binary from builder
COPY --from=builder /app/server .

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./server"]
