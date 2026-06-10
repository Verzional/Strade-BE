# Use the full Go image (we need the Go tools to run the code)
FROM golang:1.21-alpine

WORKDIR /app

# Copy your source code and routes into the container
COPY main.go ./
COPY routes ./routes

# Initialize the Go module
RUN go mod init api-gateway

# Expose the gateway port
EXPOSE 8080

# Run the Go file immediately!
CMD ["go", "run", "main.go"]
