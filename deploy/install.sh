#!/bin/bash
# RVCinemaView Installation Script for RISC-V Linux

set -e

INSTALL_DIR="/opt/rvcinemaview"
SERVICE_NAME="rvcinemaview"
USER_NAME="rvcinemaview"

echo "=== RVCinemaView Installation ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (sudo)"
    exit 1
fi

# Create system user
if ! id "$USER_NAME" &>/dev/null; then
    echo "Creating system user: $USER_NAME"
    useradd -r -s /sbin/nologin -d "$INSTALL_DIR" "$USER_NAME"
fi

# Create installation directory
echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/data/thumbnails"

# Copy binary
echo "Installing binary..."
cp rvcinemaview-riscv64 "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/rvcinemaview-riscv64"

# Copy config if not exists
if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    echo "Installing default configuration..."
    cp config.example.yaml "$INSTALL_DIR/config.yaml"
    echo "IMPORTANT: Edit $INSTALL_DIR/config.yaml to configure your media paths!"
fi

# Set ownership
chown -R "$USER_NAME:$USER_NAME" "$INSTALL_DIR"

# Install systemd service
echo "Installing systemd service..."
cp rvcinemaview.service /etc/systemd/system/
systemctl daemon-reload

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "1. Edit configuration: nano $INSTALL_DIR/config.yaml"
echo "2. Add your media paths to the config"
echo "3. Enable service: systemctl enable $SERVICE_NAME"
echo "4. Start service: systemctl start $SERVICE_NAME"
echo "5. Check status: systemctl status $SERVICE_NAME"
echo "6. View logs: journalctl -u $SERVICE_NAME -f"
echo ""
echo "Server will be available at: http://<your-ip>:6540"
