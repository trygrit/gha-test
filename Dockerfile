FROM golang:1.23-alpine AS builder

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

# Install terraform, bash, and AWS CLI
RUN apk add --no-cache curl unzip bash python3 aws-cli && \
    curl -LO https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION:-1.12.1}/terraform_${TERRAFORM_VERSION:-1.12.1}_linux_amd64.zip && \
    unzip terraform_${TERRAFORM_VERSION:-1.12.1}_linux_amd64.zip -d /usr/local/bin && \
    rm terraform_${TERRAFORM_VERSION:-1.12.1}_linux_amd64.zip && \
    apk del curl unzip

# Copy the binary and templates from builder
COPY --from=builder /app/commentor /app/commentor
COPY --from=builder /app/internal/templates /app/internal/templates

# Create directory for GitHub event file
RUN mkdir -p /github/workflow

# Set the entrypoint
ENTRYPOINT ["/app/commentor"]
