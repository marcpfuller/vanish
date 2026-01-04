# Integration Tests with testcontainers-go

This directory contains end-to-end integration tests using [testcontainers-go](https://golang.testcontainers.org/).

## Overview

The integration tests verify the complete vanish workflow:
1. Start a mock NAS (SFTP server) in a Docker container
2. Connect the NAS to a Tailscale network
3. Run vanish sync to backup files over the Tailscale VPN
4. Verify files were synced correctly

## Advantages of testcontainers-go

- ✅ **Pure Go**: No shell commands or subprocess management
- ✅ **Automatic cleanup**: Containers are removed even if tests panic
- ✅ **Better wait strategies**: Built-in health checking for services
- ✅ **Parallel execution**: Multiple tests can run concurrently
- ✅ **Network isolation**: Each test gets its own network
- ✅ **IDE integration**: Better debugging support in VS Code/GoLand

## Prerequisites

### 1. Docker

Install Docker Desktop (macOS/Windows) or Docker Engine (Linux):
- macOS: https://docs.docker.com/desktop/install/mac-install/
- Windows: https://docs.docker.com/desktop/install/windows-install/
- Linux: https://docs.docker.com/engine/install/

Verify installation:
```bash
docker version
docker ps
```

### 2. Tailscale Test Auth Key

Create a Tailscale auth key for testing:

1. Go to https://login.tailscale.com/admin/settings/keys
2. Click **Generate auth key**
3. Configure the key:
   - ✅ **Ephemeral** (node is removed when it disconnects)
   - ✅ **Reusable** (can authenticate multiple times)
   - Set expiration to 90 days or more
   - Add tag: `tag:test` (optional but recommended)
4. Copy the auth key (starts with `tskey-auth-`)
5. Export it as an environment variable:
   ```bash
   export TS_TEST_AUTHKEY=tskey-auth-xxxxx
   ```

**Important**: Keep this auth key secure! Don't commit it to git.

## Running the Tests

### Build Test Docker Images

First, build the NAS container image:
```bash
cd test/integration
docker build -t vanish-test-nas:latest -f Dockerfile.nas .
```

Or use the Makefile:
```bash
make test-integration-build
```

### Run All Integration Tests

```bash
# From repository root
make test-integration

# Or directly with go test
go test -v -tags=integration ./test/integration/... -timeout=15m
```

### Run Specific Tests

```bash
# Basic NAS container test (no Tailscale required)
go test -v -tags=integration ./test/integration/... -run TestIntegration_NASContainerBasics

# Tailscale connection test (requires TS_TEST_AUTHKEY)
go test -v -tags=integration ./test/integration/... -run TestIntegration_TailscaleConnection

# Full workflow test (requires TS_TEST_AUTHKEY)
go test -v -tags=integration ./test/integration/... -run TestIntegration_FullSyncWorkflow

# Parallel execution test
go test -v -tags=integration ./test/integration/... -run TestIntegration_ParallelContainers
```

### Skip Long-Running Tests

```bash
go test -v -tags=integration -short ./test/integration/...
```

## Test Cases

### TestIntegration_NASContainerBasics
**Duration**: ~10-15 seconds  
**Requirements**: Docker only (no Tailscale)

Quick smoke test that verifies:
- Docker container builds successfully
- SSH/SFTP server starts
- Container exposes port 22
- Container logs are accessible

### TestIntegration_TailscaleConnection
**Duration**: ~60-90 seconds  
**Requirements**: Docker + `TS_TEST_AUTHKEY`

Verifies Tailscale functionality:
- Container starts with Tailscale daemon
- Authenticates with Tailscale network
- Obtains a Tailscale IP (100.x.x.x)
- Connection is stable

### TestIntegration_FullSyncWorkflow
**Duration**: ~2-3 minutes  
**Requirements**: Docker + `TS_TEST_AUTHKEY`

Complete end-to-end test:
- Creates test files to sync
- Starts NAS with Tailscale
- Configures vanish sync job
- Sets environment variables (TS_AUTHKEY, NAS_CREDS)
- TODO: Executes vanish sync
- TODO: Verifies files were synced to NAS

### TestIntegration_ParallelContainers
**Duration**: ~5 seconds  
**Requirements**: Docker only

Demonstrates parallel test execution with testcontainers-go.

## Environment Variables

### Required for Tailscale Tests
- `TS_TEST_AUTHKEY`: Tailscale ephemeral auth key

### Optional for Debugging
- `TESTCONTAINERS_RYUK_DISABLED=true`: Keep containers after test failure
- `TESTCONTAINERS_DEBUG=true`: Enable verbose testcontainers logging

## Debugging

### Keep Containers Running After Failure

```bash
TESTCONTAINERS_RYUK_DISABLED=true go test -v -tags=integration ./test/integration/...
```

This prevents automatic cleanup so you can inspect containers:
```bash
docker ps -a
docker logs <container-id>
docker exec -it <container-id> bash
```

### Enable Verbose Logging

```bash
TESTCONTAINERS_DEBUG=true go test -v -tags=integration ./test/integration/...
```

### Manual Container Testing

Build and run the NAS container manually:
```bash
cd test/integration
docker build -t vanish-test-nas:latest -f Dockerfile.nas .
docker run -it --rm \
  -e TS_AUTHKEY=$TS_TEST_AUTHKEY \
  -p 2222:22 \
  vanish-test-nas:latest
```

Test SSH connection:
```bash
ssh -p 2222 nasuser@localhost
# Password: naspass123
```

Test SFTP:
```bash
sftp -P 2222 nasuser@localhost
# Password: naspass123
sftp> ls
sftp> cd backups
sftp> pwd
```

## CI/CD Integration

The integration tests are configured to run in GitHub Actions via `.github/workflows/integration.yml`.

**Required Repository Secret**:
- `TS_TEST_AUTHKEY`: Your Tailscale test auth key

The workflow:
1. Builds the test NAS Docker image
2. Runs all integration tests
3. Reports results
4. Cleans up containers automatically

## Troubleshooting

### Docker Not Running
```
Error: Cannot connect to the Docker daemon
```
**Solution**: Start Docker Desktop or Docker daemon

### Tailscale Auth Fails
```
Error: Tailscale failed to connect
```
**Solutions**:
- Verify `TS_TEST_AUTHKEY` is set and valid
- Check auth key hasn't expired
- Ensure auth key is ephemeral and reusable
- Verify your Tailscale account is active

### Container Build Fails
```
Error: failed to start container
```
**Solutions**:
- Check Docker has enough resources (memory/disk)
- Verify Dockerfile.nas is present
- Try building manually: `docker build -f Dockerfile.nas .`
- Check Docker logs for specific error

### Tests Timeout
```
Error: timeout waiting for Tailscale connection
```
**Solutions**:
- Increase timeout: `-timeout=20m`
- Check network connectivity
- Verify Tailscale service is operational
- Run without Tailscale: `-run TestIntegration_NASContainerBasics`

### Permission Denied (Linux)
```
Error: permission denied while trying to connect to Docker daemon
```
**Solution**: Add user to docker group
```bash
sudo usermod -aG docker $USER
newgrp docker
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Integration Test                       │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  testcontainers-go                               │  │
│  │  • Manages container lifecycle                   │  │
│  │  • Provides wait strategies                      │  │
│  │  • Handles cleanup automatically                 │  │
│  └────────────────┬─────────────────────────────────┘  │
│                   │                                      │
│                   ▼                                      │
│  ┌──────────────────────────────────────────────────┐  │
│  │  NAS Container (Dockerfile.nas)                  │  │
│  │  ┌────────────────────────────────────────────┐ │  │
│  │  │  Ubuntu 24.04                              │ │  │
│  │  │  • OpenSSH Server (SFTP)                   │ │  │
│  │  │  • Tailscale Client                        │ │  │
│  │  │  • Test user: nasuser / naspass123         │ │  │
│  │  │  • Backup directory: /home/nasuser/backups │ │  │
│  │  └────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────┘  │
│                   │                                      │
│                   │ Tailscale VPN                        │
│                   ▼                                      │
│  ┌──────────────────────────────────────────────────┐  │
│  │  vanish CLI (test execution)                     │  │
│  │  • Uses environment variables                    │  │
│  │  • Connects via Tailscale IP                     │  │
│  │  • Syncs files over SFTP                         │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Future Enhancements


- [ ] Network failure simulation
- [ ] Verify file integrity after sync
