FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Use a smaller base image for the final container
FROM alpine:3.19

WORKDIR /app

# Install terraform
RUN apk add --no-cache curl unzip && \
    curl -LO https://releases.hashicorp.com/terraform/1.7.0/terraform_1.7.0_linux_amd64.zip && \
    unzip terraform_1.7.0_linux_amd64.zip -d /usr/local/bin && \
    rm terraform_1.7.0_linux_amd64.zip && \
    apk del curl unzip

# Copy the binary from builder
COPY --from=builder /app/commentor /app/commentor

# Create directory for GitHub event file
RUN mkdir -p /github/workflow

# Set the entrypoint
ENTRYPOINT ["/app/commentor"]
