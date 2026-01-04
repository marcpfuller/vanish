//go:build integration
// +build integration

package network_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTsnetProvider_Integration_FullWorkflow(t *testing.T) {
	authKey := os.Getenv("TS_TEST_AUTHKEY")
	if authKey == "" {
		t.Skip("TS_TEST_AUTHKEY not set, skipping integration test")
	}

	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	// Start the provider
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := provider.Start(ctx, authKey, "vanish-net-test")
	require.NoError(t, err, "Failed to start Tailscale")

	// Get local address (should succeed after start)
	addr, err := provider.LocalAddr()
	require.NoError(t, err, "Failed to get local address")
	assert.NotEmpty(t, addr, "Local address should not be empty")
	t.Logf("Tailscale local address: %s", addr)

	// Test Dial (this will fail because we don't have a peer to dial,
	// but it should fail at connection, not because tsnet isn't started)
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()

	// Try to dial an address that doesn't exist on the tailnet
	_, err = provider.Dial(dialCtx, "tcp", "100.64.0.1:80")
	// Should get a connection error, not a "not started" error
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "not started")

	// Close should work
	err = provider.Close()
	assert.NoError(t, err)

	// After close, Dial and LocalAddr should fail
	_, err = provider.LocalAddr()
	assert.Error(t, err)

	_, err = provider.Dial(context.Background(), "tcp", "example.com:80")
	assert.Error(t, err)
}

func TestTsnetProvider_Integration_MultipleStarts(t *testing.T) {
	authKey := os.Getenv("TS_TEST_AUTHKEY")
	if authKey == "" {
		t.Skip("TS_TEST_AUTHKEY not set, skipping integration test")
	}

	provider := network.NewTsnetProvider()
	defer func() { _ = provider.Close() }() //nolint:errcheck

	// Start once
	ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel1()

	err := provider.Start(ctx1, authKey, "vanish-net-test-multi")
	require.NoError(t, err, "First start failed")

	// Get first address
	addr1, err := provider.LocalAddr()
	require.NoError(t, err)
	assert.NotEmpty(t, addr1)

	// Try to start again (should return error or be idempotent)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	err = provider.Start(ctx2, authKey, "vanish-net-test-multi-2")
	// Implementation may handle this differently:
	// - Could be an error (already started)
	// - Could be idempotent (returns nil)
	// We just verify it doesn't panic
	t.Logf("Second start result: %v", err)

	// Clean up
	err = provider.Close()
	assert.NoError(t, err)
}
