# Development image with auto-rebuild on restart
FROM docker.io/golang:tip-trixie

WORKDIR /app

# Install ffmpeg for thumbnails
RUN apt-get update && \
    apt-get install -y --no-install-recommends ffmpeg ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Expose port
EXPOSE 6540

# Project mounted at /app/src (includes config.yaml, data/, media/)
ENTRYPOINT ["bash", "-c", "cd /app/src && echo 'Building...' && go build -ldflags='-s -w' -o /tmp/rvcinemaview ./cmd/rvcinemaview && echo 'Starting...' && exec /tmp/rvcinemaview \"$@\"", "--"]
CMD ["-config", "/app/src/config.yaml"]
