# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web-app

# Copy package files
COPY web-app/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web-app/ ./

# Build frontend
RUN npm run build

# Backend build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway cmd/server/main.go

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/gateway .

# Copy frontend static files from frontend-builder stage
COPY --from=frontend-builder /app/web-app/dist ./dist

# Copy migrations
COPY --from=builder /app/internal/database/migrations ./migrations

# Copy config
COPY --from=builder /app/configs ./configs

# Set static directory for the UI (matches where we copy dist above)
ENV SERVER_STATIC_DIR=/app/dist

# Expose port
EXPOSE 8080

# Run the application
CMD ["./gateway"]
