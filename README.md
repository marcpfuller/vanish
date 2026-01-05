# vanish

A zero-footprint travel backup CLI in Go using embedded Tailscale (tsnet) and system credential storage for secure, ephemeral remote syncing.

## Features

- **Zero-Footprint**: Secrets are never written to disk; all sensitive data stays in the system keychain
- **Ephemeral VPN**: Uses Tailscale's `tsnet` in ephemeral mode for temporary, secure connections
- **System Keyring Integration**: Uses native credential storage (Windows Credential Manager, macOS Keychain, Linux keyring)
- **SFTP Sync**: Leverages `rclone` for robust file synchronization over SFTP
- **TDD Architecture**: Built with Test-Driven Development using Mockery v3.6.1 for interface mocking
- **Pure Go**: No CGO dependencies, cross-compiles easily to all platforms

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

**Installation:**

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

Store your Tailscale and NAS credentials securely in the system keychain:

```bash
vanish setup
```

This will prompt you for:
- **Tailscale auth key** (get one from: https://login.tailscale.com/admin/settings/keys)
- **NAS username and password**

Credentials are stored in your system's native credential manager and never written to disk.

### 2. Configure Sync Jobs

Copy the example configuration and customize it:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your backup sources and destinations
```

### 3. Run Sync

Execute the backup sync operation:

```bash
vanish sync
```

Alternatively, use environment variables (useful for testing):

```bash
# Linux/macOS
export TS_AUTHKEY="tskey-auth-xxxxx"
export NAS_CREDS="username:password"
vanish sync

# Windows PowerShell
$env:TS_AUTHKEY = "tskey-auth-xxxxx"
$env:NAS_CREDS = "username:password"
vanish sync
```

## Running as a Scheduled Service

Vanish can run as a background service that automatically performs backups on a schedule using cron expressions.

### 1. Configure Scheduling

Edit your `config.yaml` to enable scheduling:

```yaml
schedule:
  enabled: true
  cron: "0 2 * * *"  # Daily at 2:00 AM
```

**Cron Expression Examples:**
- `"0 2 * * *"` - Daily at 2:00 AM
- `"0 */6 * * *"` - Every 6 hours
- `"0 0 * * 0"` - Weekly on Sunday at midnight
- `"0 3 * * 1-5"` - Weekdays at 3:00 AM
- `"*/30 * * * *"` - Every 30 minutes

### 2. Run the Service

```bash
# Start the service with logging
export VANISH_LOG_FILE=/var/log/vanish/vanish.log
vanish service
```

The service will:
- ✅ Start and wait for scheduled times
- ⏰ Run sync jobs automatically at scheduled times
- 📝 Log all operations to the specified log file
- 🔄 Continue running until interrupted (Ctrl+C or SIGTERM)

### Linux (systemd)

Create a systemd service file at `/etc/systemd/system/vanish.service`:

```ini
[Unit]
Description=Vanish Scheduled Backup Service
After=network.target

[Service]
Type=simple
User=yourusername
WorkingDirectory=/home/yourusername
Environment="VANISH_LOG_FILE=/var/log/vanish/vanish.log"
ExecStart=/usr/local/bin/vanish service
Restart=on-failure
RestartSec=60

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo mkdir -p /var/log/vanish
sudo chown yourusername:yourusername /var/log/vanish
sudo systemctl enable vanish
sudo systemctl start vanish
sudo systemctl status vanish
```

View logs:

```bash
sudo journalctl -u vanish -f
# Or directly from the log file:
tail -f /var/log/vanish/vanish.log
```

### Windows (Task Scheduler or Service)

### Windows Service (NSSM)

Use NSSM (Non-Sucking Service Manager) to run vanish as a Windows service:

```powershell
# Download and install NSSM from https://nssm.cc/download
# Or use Chocolatey: choco install nssm

# Install vanish as a service
nssm install VanishBackup "C:\Program Files\vanish\vanish.exe" service
nssm set VanishBackup AppDirectory "C:\Program Files\vanish"
nssm set VanishBackup AppEnvironmentExtra VANISH_LOG_FILE=C:\ProgramData\vanish\logs\vanish.log
nssm start VanishBackup
```

View logs:

```powershell
Get-Content C:\logs\vanish.log -Tail 50 -Wait
```

### macOS (launchd)

Create a plist file at `~/Library/LaunchAgents/com.vanish.backup.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vanish.backup</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vanish</string>
        <string>service</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/yourusername</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>VANISH_LOG_FILE</key>
        <string>/Users/yourusername/Library/Logs/vanish.log</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load and start:

```bash
launchctl load ~/Library/LaunchAgents/com.vanish.backup.plist
launchctl start com.vanish.backup
```

View logs:

```bash
tail -f ~/Library/Logs/vanish.log
```

### Log Format

When `VANISH_LOG_FILE` is set, all logs include timestamps:

```
2024/01/15 14:30:00.123456 vanish starting...
2024/01/15 14:30:00.234567 Attempting to initialize system keyring...
2024/01/15 14:30:00.345678 System keyring initialized successfully
2024/01/15 14:30:00.456789 Running sync command...
```

Logs are appended to the file (never truncated), so you may want to set up log rotation.

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

## Credential Setup

Vanish stores credentials securely in your system's native credential manager (Windows Credential Manager, macOS Keychain, or Linux keyring).

### Option 1: Interactive Setup (Recommended)

Run the setup command and follow the prompts:

```bash
./bin/vanish setup
```

You'll be prompted for:
- **Tailscale Auth Key**: Get from https://login.tailscale.com/admin/settings/keys
  - Create an auth key with "Ephemeral" and appropriate expiration
- **NAS Username**: Your SFTP username
- **NAS Password**: Your SFTP password

Credentials are stored securely in your system keyring.

### Option 2: Environment Variables (Development/Testing)

For development, testing, or CI/CD pipelines, you can use environment variables:

```bash
# Set environment variables
export TS_AUTHKEY="tskey-auth-xxxxx"
export NAS_CREDS="username:password"

# Run vanish commands
./bin/vanish sync
```

**When to use:**
- Development and testing
- CI/CD pipelines
- Environments without keyring access

**Security Warning**: Environment variables are less secure than system keyring. Only use this for development/testing, not production deployments.

The application automatically falls back to environment variables if credentials aren't found in the keyring.

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

This project follows Test-Driven Development (TDD) principles with comprehensive unit and integration tests.

### Unit Tests

```bash
# Run all unit tests with coverage
make test

# Run tests for a specific package
go test -v ./internal/vault/...

# Run with race detection
go test -race ./...
```

### Integration Tests

Integration tests use [testcontainers-go](https://golang.testcontainers.org/) to spin up a real SFTP server with Tailscale in Docker containers.

**Prerequisites:**
- Docker Desktop or Docker Engine installed
- Tailscale test auth key (optional, for Tailscale tests)

**Setup:**
```bash
# Build test Docker images
make test-integration-build

# Get a Tailscale test auth key (optional)
# Go to https://login.tailscale.com/admin/settings/keys
# Create an ephemeral, reusable auth key
export TS_TEST_AUTHKEY=tskey-auth-xxxxx

# Run integration tests
make test-integration
```

**What gets tested:**
- ✅ SFTP server connectivity
- ✅ Tailscale VPN connection establishment
- ✅ File synchronization over Tailscale
- ✅ Environment variable fallback for credentials
- ✅ End-to-end workflow


## Platform-Specific Notes

### Linux
- Uses system keyring (GNOME Keyring, KWallet, or KDEWallet) for secure credential storage
- If keyring is unavailable, automatically falls back to encrypted file-based storage
- File-based storage uses AES-256-GCM encryption at `~/.vanish/secrets.enc`

### macOS
- Uses macOS Keychain for credential storage
- May show security prompts when accessing keychain for the first time
- Self-signed binaries may require removing quarantine: `xattr -d com.apple.quarantine vanish`

### Windows
- Uses Windows Credential Manager for credential storage
- Compatible with both PowerShell and Command Prompt
- No additional dependencies required

### WSL (Windows Subsystem for Linux)
- System keyring (DBus) is typically unavailable in WSL environments
- Automatically detects WSL and falls back to encrypted file-based storage
- Help message displayed on first run with alternative keyring options
- Set `VANISH_USE_FILE_STORE=1` environment variable to suppress warnings
See [test/integration/README.md](test/integration/README.md) for detailed documentation.

## Security Considerations

- **No Disk Storage**: Secrets are only stored in the system keychain (keyring)
- **Ephemeral Tailscale**: The VPN connection is temporary and leaves no persistent state
- **Minimal Dependencies**: Only uses well-vetted libraries for critical security functions

## Dependencies

- [zalando/go-keyring](https://github.com/zalando/go-keyring) - System keychain access
- [tailscale.com/tsnet](https://pkg.go.dev/tailscale.com/tsnet) - Embedded Tailscale VPN
- [rclone/rclone](https://github.com/rclone/rclone) - File synchronization

## License

See LICENSE file for details.
