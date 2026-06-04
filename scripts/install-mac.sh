#!/bin/bash

set -e

REPO="yourusername/115LocalNatManager"
INSTALL_DIR="$HOME/.115manager"
BIN_NAME="115manager"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_os() {
    OS="$(uname -s)"
    case "${OS}" in
        Linux*)     MACHINE=linux;;
        Darwin*)    MACHINE=darwin;;
        *)          MACHINE="${OS}"
    esac
}

detect_arch() {
    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64*)    ARCH=amd64;;
        arm64*)     ARCH=arm64;;
        aarch64*)   ARCH=arm64;;
        *)          ARCH=amd64
    esac
}

get_latest_version() {
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        print_error "Failed to get latest version"
        exit 1
    fi
    echo "$VERSION"
}

download_binary() {
    VERSION=$1
    FILENAME="${BIN_NAME}-${MACHINE}-${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
    
    print_info "Downloading ${FILENAME}..."
    
    mkdir -p "${INSTALL_DIR}"
    
    if command -v curl &> /dev/null; then
        curl -L -o "${INSTALL_DIR}/${FILENAME}" "$URL"
    elif command -v wget &> /dev/null; then
        wget -O "${INSTALL_DIR}/${FILENAME}" "$URL"
    else
        print_error "curl or wget is required"
        exit 1
    fi
    
    cd "${INSTALL_DIR}"
    tar -xzf "${FILENAME}"
    rm -f "${FILENAME}"
    chmod +x "${BIN_NAME}"
}

create_launchd_plist() {
    PLIST_PATH="$HOME/Library/LaunchAgents/com.115manager.plist"
    
    cat > "${PLIST_PATH}" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.115manager</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BIN_NAME}</string>
        <string>-data</string>
        <string>${INSTALL_DIR}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${INSTALL_DIR}/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>${INSTALL_DIR}/stderr.log</string>
</dict>
</plist>
EOF
    
    print_info "Created launchd plist at ${PLIST_PATH}"
}

create_systemd_service() {
    SERVICE_PATH="$HOME/.config/systemd/user/115manager.service"
    
    mkdir -p "$(dirname ${SERVICE_PATH})"
    
    cat > "${SERVICE_PATH}" << EOF
[Unit]
Description=115 Local NAT Manager
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BIN_NAME} -data ${INSTALL_DIR}
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
EOF
    
    print_info "Created systemd service at ${SERVICE_PATH}"
}

setup_auto_start() {
    case "${MACHINE}" in
        darwin)
            create_launchd_plist
            launchctl load "${PLIST_PATH}"
            print_info "Service loaded via launchd"
            ;;
        linux)
            create_systemd_service
            systemctl --user daemon-reload
            systemctl --user enable 115manager
            systemctl --user start 115manager
            print_info "Service enabled via systemd"
            ;;
    esac
}

print_success() {
    echo ""
    echo "=========================================="
    echo -e "${GREEN}Installation Complete!${NC}"
    echo "=========================================="
    echo ""
    echo "Binary installed to: ${INSTALL_DIR}/${BIN_NAME}"
    echo "Data directory: ${INSTALL_DIR}"
    echo "Server port: 11580"
    echo ""
    echo "Commands:"
    echo "  Start:   ${INSTALL_DIR}/${BIN_NAME}"
    echo "  Stop:    Ctrl+C or kill the process"
    echo ""
    echo "The service will start automatically on boot."
    echo ""
    echo "Chrome Extension:"
    echo "  1. Open chrome://extensions"
    echo "  2. Enable Developer mode"
    echo "  3. Click 'Load unpacked'"
    echo "  4. Select the extension folder"
    echo ""
}

main() {
    echo "=========================================="
    echo "  115 Local NAT Manager Installer"
    echo "=========================================="
    echo ""
    
    detect_os
    detect_arch
    print_info "Detected OS: ${MACHINE}, Architecture: ${ARCH}"
    
    VERSION=$(get_latest_version)
    print_info "Latest version: ${VERSION}"
    
    download_binary "${VERSION}"
    
    setup_auto_start
    
    print_success
}

main "$@"
