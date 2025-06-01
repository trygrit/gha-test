FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o commentor ./cmd/commentor

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Create directory for GitHub event file
RUN mkdir -p /github/workflow

# Copy the binary from builder
COPY --from=builder /app/commentor .

# Set the entrypoint
ENTRYPOINT ["/app/commentor"]
