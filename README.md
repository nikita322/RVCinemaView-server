# Cinema View Server

Lightweight media server for RISC-V devices (Orange Pi RV2) and other platforms.

## Features

- **Lightweight** - Pure Go, no CGO, minimal dependencies
- **RISC-V Support** - Designed for Orange Pi RV2 and similar devices
- **Direct Play** - HTTP Range streaming, no transcoding
- **Thumbnails** - Auto-generated video previews (requires ffmpeg)
- **Progress Tracking** - Resume playback from where you left off
- **Library Tree** - Single API call returns entire folder structure

## Quick Start

### 1. Download or Build

**Download binary:**
```bash
wget https://github.com/user/cinemaview-server/releases/latest/download/cinemaview-riscv64
chmod +x cinemaview-riscv64
```

**Or build from source:**
```bash
# For local testing
go build -o cinemaview ./cmd/rvcinemaview

# For RISC-V deployment
GOOS=linux GOARCH=riscv64 go build -o cinemaview-riscv64 ./cmd/rvcinemaview
```

### 2. Configure

```bash
cat > config.yaml << EOF
server:
  host: "0.0.0.0"
  port: 6540

library:
  path: "/path/to/your/media"
  name: "My Media"

database:
  path: "data/library.db"

thumbnails:
  output_dir: "data/thumbnails"
EOF
```

### 3. Run

```bash
./cinemaview -config config.yaml
```

## Systemd Service

```bash
cd deploy/
sudo ./install.sh
sudo nano /opt/rvcinemaview/config.yaml  # Configure your media path
sudo systemctl enable rvcinemaview
sudo systemctl start rvcinemaview
```

## Docker / Podman

```bash
podman build -t cinemaview-server .
podman run -d \
  -p 6540:6540 \
  -v /path/to/media:/media:ro \
  -v ./data:/app/data \
  cinemaview-server
```

## Requirements

- RISC-V 64-bit Linux (or any Go-supported platform)
- Optional: ffmpeg/ffprobe for thumbnails and metadata extraction

## Configuration Reference

```yaml
server:
  host: "0.0.0.0"          # Listen address
  port: 6540               # Listen port
  read_timeout: 30s        # Request read timeout
  write_timeout: 0s        # Response write timeout (0 = unlimited for streaming)

library:
  path: "/media/movies"    # Media directory to scan
  name: "Media Library"    # Display name for the library

database:
  path: "data/library.db"  # SQLite database path

thumbnails:
  output_dir: "data/thumbnails"  # Thumbnail cache directory
  cache_capacity: 1000           # Max items in memory
  cache_max_size: 536870912      # Max memory usage (512 MB)

logging:
  level: "info"            # Log level: debug, info, warn, error
  pretty: true             # Human-readable logs
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/library/tree` | Get full library structure |
| POST | `/api/v1/library/scan` | Trigger library rescan |
| GET | `/api/v1/media/{id}` | Get media details and stream URL |
| GET | `/api/v1/media/{id}/stream` | Stream media file (HTTP Range) |
| GET | `/api/v1/media/{id}/thumbnail` | Get video thumbnail |
| POST | `/api/v1/playback/{id}/position` | Save playback position |
| GET | `/api/v1/playback/{id}/position` | Get playback position |
| GET | `/api/v1/playback/continue` | Get continue watching list |

## Supported Video Formats

Any format supported by the client player:
- Containers: MP4, MKV, AVI, MOV, WebM
- Video: H.264, H.265/HEVC, VP8, VP9, AV1
- Audio: AAC, AC3, EAC3, DTS, MP3, FLAC, Opus

## License

MIT
