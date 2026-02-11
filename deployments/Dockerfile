# Build Stage for Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Copy frontend dependency files
COPY frontend/package*.json ./
# Install dependencies (ignoring scripts to avoid potential platform issues)
RUN npm ci --ignore-scripts

# Copy frontend source
COPY frontend/ .

# Build frontend
RUN npm run build

# Build Stage for Backend
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app/backend

# Install build dependencies
RUN apk add --no-cache git make build-base

# Copy backend dependency files
COPY backend/go.* ./
RUN go mod download

# Copy backend source
COPY backend/ .

# Build backend binaries
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/build/takhin ./cmd/takhin
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/build/takhin-cli ./cmd/takhin-cli

# Final System Image
FROM alpine:3.19
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata bash curl

# Create user for security
RUN addgroup -S takhin && adduser -S takhin -G takhin

# Create data directories
RUN mkdir -p /data/takhin /app/configs /app/static && \
    chown -R takhin:takhin /data /app

# Copy artifacts from builders
COPY --from=backend-builder /app/build/takhin /app/takhin
COPY --from=backend-builder /app/build/takhin-cli /usr/local/bin/takhin-cli
COPY --from=frontend-builder /app/frontend/dist /app/static

# Copy configuration (will be mounted or copied via render_config script)
# We expect the config file to be at /app/configs/takhin.yaml

# Expose ports
# 9092: Kafka Protocol
# 9090: Metrics
# 9091: Health Check
# 8080: Web UI (served by takhin binary if configured, or separate nginx)
EXPOSE 9092 9090 9091 8080

# Switch to non-root user
USER takhin

# Volume for data
VOLUME ["/data"]

# Entrypoint
ENTRYPOINT ["/app/takhin"]
CMD ["-config", "/app/configs/takhin.yaml"]
