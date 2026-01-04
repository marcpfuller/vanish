//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/cmd"
	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/bdkmv/vanish/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	nasUsername = "nasuser"
	nasPassword = "naspass123"
)

// NASContainer wraps the SFTP+Tailscale test container
type NASContainer struct {
	testcontainers.Container
	Host     string
	SSHPort  int
	Username string
	Password string
}

// SetupNASContainer creates and starts the mock NAS with SFTP and Tailscale
func SetupNASContainer(ctx context.Context, t *testing.T, tsAuthKey string) (*NASContainer, error) {
	// Build the container from our Dockerfile
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "./",
			Dockerfile:    "Dockerfile.nas",
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"22/tcp"},
		Env: map[string]string{
			"TS_AUTHKEY": tsAuthKey,
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("Starting SSH server").WithStartupTimeout(90*time.Second),
			wait.ForListeningPort("22/tcp").WithStartupTimeout(90*time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "22")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	nasContainer := &NASContainer{
		Container: container,
		Host:      host,
		SSHPort:   mappedPort.Int(),
		Username:  nasUsername,
		Password:  nasPassword,
	}

	// Log container information
	t.Logf("NAS container started at %s:%d (user: %s)", host, mappedPort.Int(), nasUsername)

	return nasContainer, nil
}

// GetTailscaleIP returns the Tailscale IP of the NAS container
func (c *NASContainer) GetTailscaleIP(ctx context.Context) (string, error) {
	code, reader, err := c.Exec(ctx, []string{"tailscale", "ip", "-4"})
	if err != nil {
		return "", fmt.Errorf("failed to execute tailscale ip: %w", err)
	}
	if code != 0 {
		return "", fmt.Errorf("tailscale ip exited with code %d", code)
	}

	// Read output and remove any control characters
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, reader); err != nil {
		return "", fmt.Errorf("failed to read tailscale ip output: %w", err)
	}

	// Extract only the IP (remove control chars and whitespace)
	ip := strings.TrimSpace(buf.String())
	if ip == "" {
		return "", fmt.Errorf("tailscale ip returned empty string")
	}

	// Filter out any control characters
	cleanIP := make([]byte, 0, len(ip))
	for i := 0; i < len(ip); i++ {
		ch := ip[i]
		// Keep only printable characters (space through ~) and specifically IP-related chars
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == ':' {
			cleanIP = append(cleanIP, ch)
		}
	}

	result := string(cleanIP)
	if result == "" {
		return "", fmt.Errorf("tailscale ip returned invalid data: %q", ip)
	}

	return result, nil
}

// WaitForTailscale waits for Tailscale to be fully connected
func (c *NASContainer) WaitForTailscale(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	attemptNum := 0
	for {
		select {
		case <-ctx.Done():
			// Get logs for debugging before returning error
			logs, _ := c.GetLogs(ctx)
			return fmt.Errorf("timeout waiting for Tailscale connection after %d attempts: %w\n\nContainer logs:\n%s",
				attemptNum, ctx.Err(), logs)
		case <-ticker.C:
			attemptNum++

			// First check if tailscale status works
			code, reader, err := c.Exec(ctx, []string{"tailscale", "status"})
			if err != nil {
				continue // Can't exec, keep waiting
			}

			// Read status output for debugging
			buf := new(strings.Builder)
			io.Copy(buf, reader)
			statusOutput := buf.String()

			if code == 0 {
				// Verify we can get an IP
				ip, err := c.GetTailscaleIP(ctx)
				if err == nil && ip != "" {
					return nil
				}
			}

			// Log progress every 5 attempts (15 seconds)
			if attemptNum%5 == 0 {
				fmt.Printf("  Still waiting for Tailscale (attempt %d)... Status output: %s\n",
					attemptNum, strings.TrimSpace(statusOutput))
			}
		}
	}
}

// GetLogs returns the container logs for debugging
func (c *NASContainer) GetLogs(ctx context.Context) (string, error) {
	reader, err := c.Logs(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, reader); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func TestIntegration_NASContainerBasics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start the NAS container without Tailscale for basic functionality test
	nasContainer, err := SetupNASContainer(ctx, t, "")
	require.NoError(t, err, "Failed to start NAS container")
	defer func() {
		if err := nasContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	t.Logf("NAS container started successfully at %s:%d", nasContainer.Host, nasContainer.SSHPort)

	// Verify SSH is listening
	assert.NotEmpty(t, nasContainer.Host)
	assert.Greater(t, nasContainer.SSHPort, 0)

	// Print logs for debugging
	logs, err := nasContainer.GetLogs(ctx)
	if err == nil {
		t.Logf("Container logs:\n%s", logs)
	}
}

func TestIntegration_TailscaleConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check for Tailscale test auth key
	tsAuthKey := os.Getenv("TS_TEST_AUTHKEY")
	if tsAuthKey == "" {
		t.Skip("TS_TEST_AUTHKEY not set, skipping Tailscale integration test")
	}

	ctx := context.Background()

	// Start NAS container with Tailscale
	t.Log("Starting NAS container with Tailscale...")
	nasContainer, err := SetupNASContainer(ctx, t, tsAuthKey)
	require.NoError(t, err, "Failed to start NAS container")
	defer func() {
		if err := nasContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Wait for Tailscale to connect
	t.Log("Waiting for Tailscale to connect (this may take 30-90 seconds)...")
	err = nasContainer.WaitForTailscale(ctx, 120*time.Second)
	require.NoError(t, err, "Tailscale failed to connect")

	// Get Tailscale IP
	tsIP, err := nasContainer.GetTailscaleIP(ctx)
	require.NoError(t, err, "Failed to get Tailscale IP")
	t.Logf("✓ NAS Tailscale IP: %s", tsIP)

	// Verify IP format
	assert.NotEmpty(t, tsIP)
	assert.Contains(t, tsIP, "100.") // Tailscale IPs start with 100.x
}

func TestIntegration_FullSyncWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check for Tailscale test auth key
	tsAuthKey := os.Getenv("TS_TEST_AUTHKEY")
	if tsAuthKey == "" {
		t.Skip("TS_TEST_AUTHKEY not set, skipping full integration test")
	}

	ctx := context.Background()

	// Setup test environment
	testDir := t.TempDir()
	sourceDir := filepath.Join(testDir, "source")

	// Create test files to sync
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))
	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := []byte("integration test content at " + time.Now().String())
	require.NoError(t, os.WriteFile(testFile, testContent, 0o644))

	// Start NAS container with Tailscale
	t.Log("Starting NAS container with Tailscale...")
	nasContainer, err := SetupNASContainer(ctx, t, tsAuthKey)
	require.NoError(t, err, "Failed to start NAS container")
	defer func() {
		// Print logs before terminating (helpful for debugging)
		if logs, err := nasContainer.GetLogs(ctx); err == nil {
			t.Logf("NAS Container logs:\n%s", logs)
		}
		if err := nasContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Wait for Tailscale to connect (can take up to 2 minutes for authentication + network setup)
	t.Log("Waiting for Tailscale to connect...")
	err = nasContainer.WaitForTailscale(ctx, 120*time.Second)
	require.NoError(t, err, "Tailscale failed to connect")

	// Get Tailscale IP
	tsIP, err := nasContainer.GetTailscaleIP(ctx)
	require.NoError(t, err, "Failed to get Tailscale IP")
	t.Logf("✓ NAS Tailscale IP: %s (bytes: %v)", tsIP, []byte(tsIP))

	// Create config for vanish
	cfg := &config.Config{
		Version: "1.0",
		SyncJobs: []config.SyncJob{
			{
				Name:        "Integration Test Backup",
				Source:      sourceDir,
				Destination: fmt.Sprintf("sftp://%s:%s@%s:22/backups/test", nasContainer.Username, nasContainer.Password, tsIP),
				Enabled:     true,
			},
		},
	}
	configPath := filepath.Join(testDir, "config.yaml")
	require.NoError(t, config.Save(configPath, cfg))
	t.Logf("✓ Config file created at: %s", configPath)

	// Set environment variables for vanish to use
	require.NoError(t, os.Setenv("TS_AUTHKEY", tsAuthKey))
	require.NoError(t, os.Setenv("NAS_CREDS", fmt.Sprintf("%s:%s", nasContainer.Username, nasContainer.Password)))
	defer func() {
		//nolint:errcheck // Best effort cleanup
		os.Unsetenv("TS_AUTHKEY")
		//nolint:errcheck // Best effort cleanup
		os.Unsetenv("NAS_CREDS")
	}()

	// Create vault service with file store
	vaultDir := filepath.Join(testDir, ".vanish")
	store, err := vault.NewFileStore(vaultDir)
	require.NoError(t, err, "Failed to create file store")
	vaultSvc := vault.NewVaultService(store)
	defer vaultSvc.Close()

	// Create Tailscale provider
	tsProvider := network.NewTsnetProvider()
	defer tsProvider.Close()

	// Create sync service
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	// Execute the sync!
	t.Log("🔄 Executing vanish sync command...")
	opts := cmd.SyncOptions{
		ConfigPath: configPath,
		Timeout:    2 * time.Minute,
	}

	err = cmd.Sync(vaultSvc, tsProvider, syncSvc, opts)
	if err != nil {
		// Log container output for debugging
		if logs, logErr := nasContainer.GetLogs(ctx); logErr == nil {
			t.Logf("NAS Container logs:\n%s", logs)
		}
		require.NoError(t, err, "Sync command failed")
	}

	t.Log("✓ Sync completed successfully!")

	// Verify the file was synced by checking on the NAS container
	t.Log("🔍 Verifying file was synced to NAS...")
	code, reader, err := nasContainer.Exec(ctx, []string{
		"test", "-f", "/home/nasuser/backups/test/test.txt",
	})
	require.NoError(t, err)
	if code != 0 {
		output, _ := io.ReadAll(reader)
		t.Logf("Test command output: %s", string(output))
		t.Fatalf("File not found on NAS (exit code %d)", code)
	}

	// Read the file content to verify it matches
	code, reader, err = nasContainer.Exec(ctx, []string{
		"cat", "/home/nasuser/backups/test/test.txt",
	})
	require.NoError(t, err)
	require.Equal(t, 0, code)

	syncedContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, syncedContent, "Synced file content should match source")

	t.Log("✅ Integration test passed! File successfully synced via Tailscale SFTP")
}

func TestIntegration_ParallelContainers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Verify that testcontainers-go supports parallel test execution
	t.Run("Container1", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:      "alpine:latest",
				Cmd:        []string{"sh", "-c", "echo 'Container 1' && sleep 2"},
				WaitingFor: wait.ForLog("Container 1").WithStartupTimeout(10 * time.Second),
			},
			Started: true,
		})
		require.NoError(t, err)
		defer container.Terminate(ctx)

		t.Log("✓ Container1 started and verified")
	})

	t.Run("Container2", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:      "alpine:latest",
				Cmd:        []string{"sh", "-c", "echo 'Container 2' && sleep 2"},
				WaitingFor: wait.ForLog("Container 2").WithStartupTimeout(10 * time.Second),
			},
			Started: true,
		})
		require.NoError(t, err)
		defer container.Terminate(ctx)

		t.Log("✓ Container2 started and verified")
	})
}
