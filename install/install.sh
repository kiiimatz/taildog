#!/usr/bin/env bash
set -euo pipefail

REPO="kiiimatz/taildog"
VERSION="${TAILDOG_VERSION:-latest}"
INSTALL_DIR="/usr/local/bin"
BIN_NAME="taildog"

# ── Detect OS and arch ────────────────────────────────────────────────────────
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  GOOS="linux"  ;;
  Darwin) GOOS="darwin" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64)   GOARCH="amd64" ;;
  arm64|aarch64)  GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

TARGET="${GOOS}_${GOARCH}"
echo "Detected platform: $TARGET"

# ── Resolve version ────────────────────────────────────────────────────────────
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

echo "Installing taildog $VERSION ..."

# ── Download binary ────────────────────────────────────────────────────────────
TMPDIR_LOCAL="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_LOCAL"' EXIT

ASSET="taildog_${TARGET}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

echo "Downloading $URL ..."
curl -fsSL "$URL" -o "$TMPDIR_LOCAL/$BIN_NAME"
chmod +x "$TMPDIR_LOCAL/$BIN_NAME"

# ── Install binary ─────────────────────────────────────────────────────────────
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR_LOCAL/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  sudo mv "$TMPDIR_LOCAL/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi
echo "Binary installed to $INSTALL_DIR/$BIN_NAME"

# ── systemd service (Linux only) ───────────────────────────────────────────────
if [ "$OS" = "Linux" ] && command -v systemctl &>/dev/null; then
  UNIT_FILE="/etc/systemd/system/taildog.service"
  cat <<UNIT | sudo tee "$UNIT_FILE" > /dev/null
[Unit]
Description=taildog tunnel client daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$BIN_NAME up --foreground
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT
  sudo systemctl daemon-reload
  sudo systemctl enable taildog
  echo "systemd service enabled (will start after you configure taildog)."
fi

# ── launchd agent (macOS only) ────────────────────────────────────────────────
if [ "$OS" = "Darwin" ]; then
  PLIST_DIR="$HOME/Library/LaunchAgents"
  PLIST_FILE="$PLIST_DIR/dog.tail.taildog.plist"
  mkdir -p "$PLIST_DIR"
  cat > "$PLIST_FILE" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>           <string>dog.tail.taildog</string>
  <key>ProgramArguments</key>
  <array>
    <string>$INSTALL_DIR/$BIN_NAME</string>
    <string>up</string>
    <string>--foreground</string>
  </array>
  <key>RunAtLoad</key>       <true/>
  <key>KeepAlive</key>       <true/>
  <key>StandardErrorPath</key> <string>/tmp/taildog.err</string>
  <key>StandardOutPath</key>   <string>/tmp/taildog.out</string>
</dict>
</plist>
PLIST
  launchctl load -w "$PLIST_FILE"
  echo "launchd agent loaded."
fi

echo ""
echo "taildog $VERSION installed."
echo ""
echo "  Configure relay:  $BIN_NAME configure"
echo "  Start daemon:     $BIN_NAME up"
echo "  Run relay server: $BIN_NAME server"
