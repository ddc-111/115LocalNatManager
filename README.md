# 115 Local NAT Manager

A local service + Chrome extension for managing 115 cloud downloads with automatic magnet link detection.

## Features

- **Magnet Link Detection**: Automatically detects magnet links on any webpage
- **One-Click Cloud Download**: Send magnet links directly to 115 cloud
- **Auto Download Monitor**: Automatically downloads completed files to local storage
- **File Management**: Browse, create folders, delete files in your 115 cloud
- **Token Management**: Secure refresh token based authentication
- **Cross-Platform**: Works on macOS, Windows, and Linux

## Quick Start

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/ddc-111/115LocalNatManager/main/scripts/install-mac.sh | bash
```

### Windows (PowerShell as Administrator)

```powershell
irm https://raw.githubusercontent.com/ddc-111/115LocalNatManager/main/scripts/install-windows.ps1 | iex
```

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/ddc-111/115LocalNatManager/releases)
2. Extract to a directory
3. Run the binary: `./115manager`

## Chrome Extension Setup

1. Open Chrome and go to `chrome://extensions`
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select the `extension` folder from this repository

## Configuration

### Setting Up Refresh Token

1. Get your refresh token from [115 Open Platform](https://open.115.com/)
2. Click the extension icon in Chrome
3. Go to Settings (gear icon)
4. Enter your refresh token and click "Save Token"

### Download Settings

- **Download Directory**: Where completed files are saved locally
- **Monitor Interval**: How often to check for completed downloads (default: 30 seconds)

## API Reference

The backend server runs on `http://localhost:11580` by default.

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/token` | Set refresh token |
| GET | `/api/token` | Get token status |

### File Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/files` | List files |
| GET | `/api/files/:id` | Get file info |
| PUT | `/api/files/:id` | Rename file |
| POST | `/api/files/delete` | Delete files |
| POST | `/api/files/move` | Move files |
| GET | `/api/files/search` | Search files |
| POST | `/api/folders` | Create folder |

### Cloud Download

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/download` | Add download task |
| GET | `/api/download/tasks` | List download tasks |
| DELETE | `/api/download/tasks/:hash` | Delete task |
| POST | `/api/download/clear` | Clear tasks |
| GET | `/api/download/quota` | Get download quota |
| GET | `/api/download/monitor` | Get monitor status |
| POST | `/api/download/monitor` | Toggle monitor |

### Configuration

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/config` | Get configuration |
| PUT | `/api/config` | Update configuration |

## Development

### Prerequisites

- Go 1.21+
- Chrome browser

### Build from Source

```bash
# Clone repository
git clone https://github.com/ddc-111/115LocalNatManager.git
cd 115LocalNatManager

# Build backend
cd backend
go build -o ../dist/115manager .

# Run
./dist/115manager
```

### Project Structure

```
115LocalNatManager/
├── backend/                    # Go backend service
│   ├── api/                    # 115 API client
│   ├── config/                 # Configuration management
│   ├── handler/                # HTTP handlers
│   ├── service/                # Business logic
│   └── main.go                 # Entry point
├── extension/                  # Chrome extension
│   ├── content/                # Content scripts (magnet detection)
│   ├── background/             # Background service worker
│   ├── popup/                  # Popup UI
│   └── options/                # Settings page
├── scripts/                    # Installation scripts
└── .github/workflows/          # CI/CD
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request
