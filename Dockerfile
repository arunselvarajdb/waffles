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

# Copy built frontend from frontend-builder stage to cmd/server/dist for embedding
COPY --from=frontend-builder /app/web-app/dist ./cmd/server/dist

# Build the application (this will embed the frontend dist files)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/gateway .

# Copy migrations
COPY --from=builder /app/internal/database/migrations ./migrations

# Copy config
COPY --from=builder /app/configs ./configs

# Expose port
EXPOSE 8080

# Run the application
CMD ["./gateway"]
