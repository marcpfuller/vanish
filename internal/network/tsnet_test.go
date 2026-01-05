package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestTsnetProvider(t *testing.T) {
	tests := []struct {
		name      string
		skipShort bool
		run       func(t *testing.T)
	}{
		{
			name:      "new provider",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				assert.NotNil(t, provider)

				err := provider.Close()
				assert.NoError(t, err)
			},
		},
		{
			name:      "start with invalid auth key",
			skipShort: true,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := provider.Start(ctx, "invalid-key", "test-host")
				assert.Error(t, err)
			},
		},
		{
			name:      "dial before start",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				ctx := context.Background()

				_, err := provider.Dial(ctx, "tcp", "example.com:80")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not started")
			},
		},
		{
			name:      "local addr before start",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				_, err := provider.LocalAddr()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not started")
			},
		},
		{
			name:      "multiple close",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()

				err := provider.Close()
				assert.NoError(t, err)

				err = provider.Close()
				assert.NoError(t, err)
			},
		},
		{
			name:      "close without start",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()

				err := provider.Close()
				assert.NoError(t, err)
			},
		},
		{
			name:      "start after close",
			skipShort: true,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()

				err := provider.Close()
				assert.NoError(t, err)

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				err = provider.Start(ctx, "invalid-key", "test-host")
				assert.Error(t, err)
			},
		},
		{
			name:      "start with empty hostname",
			skipShort: true,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				err := provider.Start(ctx, "invalid-key", "")
				assert.Error(t, err)
			},
		},
		{
			name:      "local addr not started",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				addr, err := provider.LocalAddr()
				assert.Error(t, err)
				assert.Empty(t, addr)
				assert.Contains(t, err.Error(), "not started")
			},
		},
		{
			name:      "dial not started",
			skipShort: false,
			run: func(t *testing.T) {
				provider := network.NewTsnetProvider()
				defer func() { _ = provider.Close() }() //nolint:errcheck

				ctx := context.Background()
				conn, err := provider.Dial(ctx, "tcp", "example.com:80")
				assert.Error(t, err)
				assert.Nil(t, conn)
				assert.Contains(t, err.Error(), "not started")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipShort && testing.Short() {
				t.Skip("Skipping Tailscale integration test in short mode")
			}
			tt.run(t)
		})
	}
}
