# Use the official Go image as the base image
FROM golang:1.18

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules files
COPY server/go.mod server/go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY server/ ./

# Build the application
RUN go build -o server server.go game.go

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./server"]
