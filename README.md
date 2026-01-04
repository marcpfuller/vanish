# vanish

A zero-footprint travel backup CLI in Go using embedded Tailscale (tsnet) and Bitwarden for secure, ephemeral remote syncing.

## Features

- **Zero-Footprint**: Secrets are never written to disk; all sensitive data stays in the system keychain
- **Ephemeral VPN**: Uses Tailscale's `tsnet` in ephemeral mode for temporary, secure connections
- **Bitwarden Integration**: Securely stores and retrieves Tailscale auth keys and NAS credentials
- **SFTP Sync**: Leverages `rclone` for robust file synchronization over SFTP
- **TDD Architecture**: Built with Test-Driven Development using Mockery v3.6.1 for interface mocking

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/bdkmv/vanish/releases) page:

| Platform | Binary |
|----------|--------|
| Linux (amd64) | `vanish-linux-amd64` |
| macOS (amd64) | `vanish-darwin-amd64` |
| Windows (amd64) | `vanish-windows-amd64.exe` |

#### Linux

```bash
# Download and install
curl -L -o vanish https://github.com/bdkmv/vanish/releases/latest/download/vanish-linux-amd64
chmod +x vanish
sudo mv vanish /usr/local/bin/
```

#### macOS

```bash
# Download and install
curl -L -o vanish https://github.com/bdkmv/vanish/releases/latest/download/vanish-darwin-amd64
chmod +x vanish
sudo mv vanish /usr/local/bin/

# If you get a security warning, allow it in System Settings > Privacy & Security
```

#### Windows

1. Download `vanish-windows-amd64.exe` from the releases page
2. Rename to `vanish.exe` (optional)
3. Move to a directory in your PATH, or run directly:

```powershell
# Run directly
.\vanish-windows-amd64.exe setup

# Or add to PATH and run
vanish setup
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/bdkmv/vanish.git
cd vanish

# Build for current platform
make build-local

# Build for all platforms (requires Docker for cross-compilation)
make build-all-docker

# The binaries will be in ./bin/
```

## Quick Start

### 1. Initial Setup

Store your Bitwarden access token securely in the system keychain:

```bash
./bin/vanish setup
```

This will prompt you for your Bitwarden Access Token and store it securely.

### 2. Configure Sync Jobs

Copy the example configuration and customize it:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your backup sources and destinations
```

### 3. Run Sync

Execute the backup sync operation:

```bash
./bin/vanish sync
```

## Platform Notes

### Linux

Works out of the box. Uses the system keyring (libsecret/GNOME Keyring or KWallet).

### macOS

Uses the macOS Keychain for secure credential storage. You may need to allow the app in System Settings if you see a security warning on first run.

### Windows

Uses Windows Credential Manager for secure storage. Run from PowerShell or Command Prompt:

```powershell
.\vanish.exe setup
.\vanish.exe sync
```

### WSL (Windows Subsystem for Linux)

If you're running in WSL, the application will automatically detect this and provide guidance:

**Option 1: Use encrypted file storage (automatic fallback)**
```bash
./bin/vanish setup
```

**Option 2: Use file storage explicitly**
```bash
export VANISH_USE_FILE_STORE=1
./bin/vanish setup
```

**Option 3: Enable DBus for system keyring**
```bash
sudo apt install dbus-x11
eval $(dbus-launch --sh-syntax)
./bin/vanish setup
```

Note: The encrypted file storage is less secure than the system keyring but provides a working solution for WSL environments.

## Configuration

The `config.yaml` file defines your sync jobs:

```yaml
version: "1.0"
sync_jobs:
  - name: "Documents Backup"
    source: "/home/user/Documents"
    destination: "sftp://nas.tailnet:22/backups/documents"
    enabled: true
```

## Bitwarden Setup

Store these secrets in your Bitwarden vault as **Secrets** (not Login items):

1. **TS_AUTHKEY**: Your Tailscale authentication key
   - Get this from: https://login.tailscale.com/admin/settings/keys
   - Create an auth key with appropriate expiration

2. **NAS_CREDS**: Your NAS SFTP credentials
   - Format: `username:password`
   - Example: `admin:MySecurePassword123`

Note: The secret names are case-sensitive and must match exactly.

## Development

### Prerequisites

- Go 1.25.5 or later
- mockery v3.6.1
- golangci-lint

### Build Commands

```bash
# Run tests
make test

# Run linter
make lint

# Check code formatting (CI)
make fmt-check

# Format code
make fmt

# Install development tools
make install-lint
make install-mockery

# Generate mocks
make update-mocks
```

### Build Targets

```bash
# Build for current platform
make build-local

# Build for Linux (amd64)
make build-linux

# Cross-compile using Docker
make build-windows-docker  # Windows
make build-darwin-docker   # macOS
make build-all-docker      # All platforms
```

### Project Structure

```
vanish/
├── cmd/vanish/          # Main application entry point
├── internal/
│   ├── cmd/             # Command implementations
│   ├── vault/           # Keychain/secret management
│   ├── network/         # Tailscale integration
│   └── sync/            # Sync operations
├── pkg/
│   └── config/          # Configuration parsing
├── mocks/               # Generated mocks
└── config.example.yaml  # Example configuration
```

## Testing

This project follows Test-Driven Development (TDD) principles:

```bash
# Run all tests with coverage
make test

# The test output includes coverage information
```

## Security Considerations

- **No Disk Storage**: Secrets are only stored in the system keychain (keyring)
- **Ephemeral Tailscale**: The VPN connection is temporary and leaves no persistent state
- **Minimal Dependencies**: Only uses well-vetted libraries for critical security functions

## Dependencies

- [zalando/go-keyring](https://github.com/zalando/go-keyring) - System keychain access
- [tailscale.com/tsnet](https://pkg.go.dev/tailscale.com/tsnet) - Embedded Tailscale VPN
- [bitwarden/sdk-go](https://github.com/bitwarden/sdk-go) - Bitwarden secrets management
- [rclone/rclone](https://github.com/rclone/rclone) - File synchronization

## License

See LICENSE file for details.
