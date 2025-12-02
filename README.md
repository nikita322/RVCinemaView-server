# RVCinemaView

Lightweight media server for RISC-V devices (Orange Pi RV2) with Android/Android TV client.

## Features

- **Lightweight Server** - Pure Go, no CGO, minimal dependencies
- **RISC-V Support** - Designed for Orange Pi RV2 and similar devices
- **Android Client** - Works on phones, tablets, and Android TV
- **Direct Play** - HTTP Range streaming, no transcoding
- **Thumbnails** - Auto-generated video previews (requires ffmpeg)
- **Progress Tracking** - Resume playback from where you left off
- **D-Pad Navigation** - Full remote control support for TV

## Quick Start

### Server Installation (RISC-V Linux)

1. **Download the binary:**
```bash
wget https://github.com/user/rvcinemaview/releases/latest/download/rvcinemaview-riscv64
chmod +x rvcinemaview-riscv64
```

2. **Create configuration:**
```bash
cat > config.yaml << EOF
server:
  host: "0.0.0.0"
  port: 8080

library:
  paths:
    - "/path/to/your/media"

database:
  path: "data/library.db"

thumbnails:
  output_dir: "data/thumbnails"
EOF
```

3. **Run:**
```bash
./rvcinemaview-riscv64 -config config.yaml
```

### Systemd Service Installation

```bash
cd deploy/
sudo ./install.sh
sudo nano /opt/rvcinemaview/config.yaml  # Configure your media paths
sudo systemctl enable rvcinemaview
sudo systemctl start rvcinemaview
```

### Android Client

1. Install APK on your Android device or TV
2. Enter server address (e.g., `192.168.1.100:6540`)
3. Browse and watch your media

## Requirements

### Server
- RISC-V 64-bit Linux (tested on Orange Pi RV2)
- Optional: ffmpeg/ffprobe for thumbnails and metadata

### Client
- Android 7.0+ (API 24)
- Android TV with Leanback support

## Building from Source

### Server
```bash
cd server

# For local testing
go build -o rvcinemaview ./cmd/rvcinemaview

# For RISC-V deployment
GOOS=linux GOARCH=riscv64 go build -o rvcinemaview-riscv64 ./cmd/rvcinemaview
```

### Android Client
```bash
cd android
./gradlew assembleRelease
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/library` | Get library folders |
| GET | `/api/v1/folders/{id}` | Get folder details |
| GET | `/api/v1/folders/{id}/items` | Get folder media items |
| GET | `/api/v1/media/{id}` | Get media details |
| GET | `/api/v1/media/{id}/stream` | Stream media file |
| GET | `/api/v1/media/{id}/thumbnail` | Get thumbnail |
| POST | `/api/v1/playback/{id}/position` | Save playback position |
| GET | `/api/v1/playback/{id}/position` | Get playback position |
| GET | `/api/v1/playback/continue` | Get continue watching list |

## Configuration

```yaml
server:
  host: "0.0.0.0"          # Listen address
  port: 6540               # Listen port
  read_timeout: 30s        # Request read timeout
  write_timeout: 0s        # Response write timeout (0 = unlimited for streaming)

library:
  paths:                   # Media directories to scan
    - "/media/movies"
    - "/media/tv"
  scan_interval: 1h        # Auto-rescan interval

database:
  path: "data/library.db"  # SQLite database path

thumbnails:
  output_dir: "data/thumbnails"  # Thumbnail cache directory
  cache_capacity: 1000           # Max items in memory
  cache_max_size: 536870912      # Max memory usage (512 MB)

logging:
  level: "info"            # Log level: debug, info, warn, error
  pretty: false            # Human-readable logs
```

## License

MIT
