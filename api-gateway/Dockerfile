# Use a Go image with the correct version
FROM golang:1.21-alpine

# Set working directory for the project
WORKDIR /go/src/github.com/order-management

# Install necessary build tools
RUN apk add --no-cache git gcc musl-dev

# Copy the proto directory first
COPY proto/ /go/src/github.com/order-management/proto/

# Copy the service code
COPY api-gateway/ /go/src/github.com/order-management/api-gateway/

# Set working directory to the service directory
WORKDIR /go/src/github.com/order-management/api-gateway

# Build the application
RUN go build -o api-gateway .

# Use a smaller base image for the final container
FROM alpine:latest

# Install CA certificates for any HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=0 /go/src/github.com/order-management/api-gateway/api-gateway .

# Expose the port the service runs on
EXPOSE 8080

# Command to run when the container starts
CMD ["./api-gateway"]
