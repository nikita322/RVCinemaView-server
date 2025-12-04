# Build stage
FROM docker.io/golang:tip-trixie AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rvcinemaview ./cmd/rvcinemaview

# Runtime stage
FROM docker.io/debian:trixie-slim

WORKDIR /app

# Install ffmpeg for thumbnails (optional but recommended)
RUN apt-get update && \
    apt-get install -y --no-install-recommends ffmpeg ca-certificates wget && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 rvcinemaview && \
    useradd -u 1000 -g rvcinemaview -s /bin/sh -m rvcinemaview

# Create data directories
RUN mkdir -p /app/data/thumbnails && \
    chown -R rvcinemaview:rvcinemaview /app

# Copy binary from builder
COPY --from=builder /build/rvcinemaview /app/rvcinemaview

# Switch to non-root user
USER rvcinemaview

# Expose port
EXPOSE 6540

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:6540/api/v1/health || exit 1

# Default command
ENTRYPOINT ["/app/rvcinemaview"]
CMD ["-config", "/app/config.yaml"]
