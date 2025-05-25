# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
# Statically link the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/server .

# Stage 2: Create the final image
FROM alpine:latest

WORKDIR /app

# Install Python, pip, ffmpeg (for gytmdl), and ca-certificates (for HTTPS)
RUN apk add --no-cache python3 py3-pip ffmpeg ca-certificates

# Install gytmdl
RUN pip3 install --no-cache-dir --break-system-packages gytmdl
RUN pip3 install --no-cache-dir --break-system-packages colorama

# Copy the static UI files
COPY ui ./ui

# Copy the built Go application from the builder stage
COPY --from=builder /app/server .

# Create the Music directory and make it writable
# The application also tries to create this, but good to have it here
RUN mkdir -p /app/Music && chmod -R 777 /app/Music

# Command to run the application
CMD ["/app/server"]
