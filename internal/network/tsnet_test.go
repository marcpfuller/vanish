package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestNewTsnetProvider(t *testing.T) {
	provider := network.NewTsnetProvider()
	assert.NotNil(t, provider)

	// Close should work even without starting
	err := provider.Close()
	assert.NoError(t, err)
}

func TestTsnetProvider_Start_InvalidAuthKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Tailscale integration test in short mode")
	}

	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail with invalid auth key
	err := provider.Start(ctx, "invalid-key", "test-host")
	assert.Error(t, err)
}

func TestTsnetProvider_DialBeforeStart(t *testing.T) {
	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	ctx := context.Background()

	// Should fail when not started
	_, err := provider.Dial(ctx, "tcp", "example.com:80")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestTsnetProvider_LocalAddrBeforeStart(t *testing.T) {
	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	// Should fail when not started
	_, err := provider.LocalAddr()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestTsnetProvider_MultipleClose(t *testing.T) {
	provider := network.NewTsnetProvider()

	// First close
	err := provider.Close()
	assert.NoError(t, err)

	// Second close should also work
	err = provider.Close()
	assert.NoError(t, err)
}

func TestTsnetProvider_CloseWithoutStart(t *testing.T) {
	provider := network.NewTsnetProvider()

	// Closing without starting should not error
	err := provider.Close()
	assert.NoError(t, err)
}

func TestTsnetProvider_StartAfterClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Tailscale integration test in short mode")
	}

	provider := network.NewTsnetProvider()

	// Close first
	err := provider.Close()
	assert.NoError(t, err)

	// Try to start after close - should fail with invalid key or timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = provider.Start(ctx, "invalid-key", "test-host")
	// Should fail - either due to being closed or invalid key
	assert.Error(t, err)
}

func TestTsnetProvider_StartWithEmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Tailscale integration test in short mode")
	}

	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Empty hostname should still work (will use default)
	err := provider.Start(ctx, "invalid-key", "")
	assert.Error(t, err) // Will fail due to invalid key, not hostname
}

func TestTsnetProvider_LocalAddrNotStarted(t *testing.T) {
	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	addr, err := provider.LocalAddr()
	assert.Error(t, err)
	assert.Empty(t, addr)
	assert.Contains(t, err.Error(), "not started")
}

func TestTsnetProvider_DialNotStarted(t *testing.T) {
	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	ctx := context.Background()
	conn, err := provider.Dial(ctx, "tcp", "example.com:80")
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "not started")
}
